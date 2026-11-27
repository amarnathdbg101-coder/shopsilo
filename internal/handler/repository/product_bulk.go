// Package repository handle db query.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"shopMe/internal/handler/dto"
	"shopMe/internal/reuse"
	"strings"
	"time"
)

func (r *ProductRepo) BulkImportProducts(ctx context.Context, shopID string, items []dto.BulkImportProductItem, updateExisting bool) (*dto.BulkImportResponse, error) {
	if len(items) == 0 {
		return &dto.BulkImportResponse{TotalRows: 0, ImportedCount: 0, UpdatedCount: 0, SkippedCount: 0, Errors: []string{}}, nil
	}

	// 1. Resolve and cache all categories
	categoryMap := make(map[string]string) // cleanName/slug -> categoryID
	var defaultCategoryID string

	rows, err := r.db.Query(ctx, `SELECT id, LOWER(TRIM(name)), LOWER(TRIM(slug)) FROM categories WHERE is_active = true`)
	if err == nil {
		for rows.Next() {
			var id, name, slug string
			if err := rows.Scan(&id, &name, &slug); err == nil {
				categoryMap[name] = id
				categoryMap[slug] = id
				if defaultCategoryID == "" {
					defaultCategoryID = id
				}
			}
		}
		rows.Close()
	}

	// Ensure a default category exists
	if defaultCategoryID == "" {
		_ = r.db.QueryRow(ctx, `
			INSERT INTO categories (name, slug, description, is_active, created_at, updated_at)
			VALUES ('General Store', 'general-store', 'Default Kirana and General Category', true, NOW(), NOW())
			ON CONFLICT (slug) WHERE shop_id IS NULL DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`).Scan(&defaultCategoryID)
		categoryMap["general store"] = defaultCategoryID
		categoryMap["general-store"] = defaultCategoryID
	}

	// Helper to resolve or auto-create category
	resolveCategory := func(catName string) string {
		clean := strings.ToLower(strings.TrimSpace(catName))
		if clean == "" {
			return defaultCategoryID
		}
		if id, found := categoryMap[clean]; found && id != "" {
			return id
		}
		slug := reuse.Slugify(clean)
		if id, found := categoryMap[slug]; found && id != "" {
			return id
		}
		if slug == "" {
			slug = fmt.Sprintf("cat-%d", time.Now().UnixNano()%1000000)
		}

		var newID string
		err := r.db.QueryRow(ctx, `
			INSERT INTO categories (name, slug, description, is_active, created_at, updated_at)
			VALUES ($1, $2, 'Auto-created from bulk import', true, NOW(), NOW())
			ON CONFLICT (slug) WHERE shop_id IS NULL DO UPDATE SET name = EXCLUDED.name
			RETURNING id
		`, strings.TrimSpace(catName), slug).Scan(&newID)
		if err == nil && newID != "" {
			categoryMap[clean] = newID
			categoryMap[slug] = newID
			return newID
		}
		return defaultCategoryID
	}

	// 2. Fetch existing products for this shop for fast conflict & upsert detection
	existingSKUs := make(map[string]string)  // lower(sku) -> productID
	existingNames := make(map[string]string) // lower(name) -> productID

	pRows, err := r.db.Query(ctx, `SELECT id, COALESCE(LOWER(TRIM(sku)), ''), LOWER(TRIM(name)) FROM products WHERE shop_id = $1 AND deleted_at IS NULL`, shopID)
	if err == nil {
		for pRows.Next() {
			var pID, sku, name string
			if err := pRows.Scan(&pID, &sku, &name); err == nil {
				if sku != "" {
					existingSKUs[sku] = pID
				}
				if name != "" {
					existingNames[name] = pID
				}
			}
		}
		pRows.Close()
	}

	// 3. Pre-validate and partition items into toInsert, toUpdate, and skipped
	type sanitizedItem struct {
		id           string
		rowNumber    int
		name         string
		slug         string
		description  string
		sku          string
		price        float64
		costPrice    float64
		comparePrice float64
		categoryID   string
		stock        int
		minStock     int
		weight       float64
		existingID   string
		imagesJSON   string
		attrsJSON    string
	}

	var toInsert []sanitizedItem
	var toUpdate []sanitizedItem
	var skippedCount int
	var errMsgs []string
	const maxTrackedErrors = 100

	addErrMsg := func(msg string) {
		if len(errMsgs) < maxTrackedErrors {
			errMsgs = append(errMsgs, msg)
		} else if len(errMsgs) == maxTrackedErrors {
			errMsgs = append(errMsgs, "... additional row warnings/errors truncated for brevity")
		}
	}

	rGenerator := rand.New(rand.NewSource(time.Now().UnixNano()))
	seenBatchSKUs := make(map[string]bool)

	for idx, item := range items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			skippedCount++
			addErrMsg(fmt.Sprintf("Row %d: Product name is empty (skipped)", idx+1))
			continue
		}
		if len(name) > 200 {
			name = name[:200]
		}

		if item.Price <= 0 {
			skippedCount++
			addErrMsg(fmt.Sprintf("Row %d ('%s'): Price must be greater than 0 (skipped)", idx+1, name))
			continue
		}

		sku := strings.TrimSpace(strings.ToUpper(item.SKU))
		if sku == "" && item.Barcode != "" {
			sku = strings.TrimSpace(strings.ToUpper(item.Barcode))
		}
		if len(sku) > 100 {
			sku = sku[:100]
		}

		cleanSKULower := strings.ToLower(sku)
		cleanNameLower := strings.ToLower(name)

		// Intra-batch duplicate check
		if cleanSKULower != "" && seenBatchSKUs[cleanSKULower] {
			skippedCount++
			addErrMsg(fmt.Sprintf("Row %d ('%s'): Duplicate SKU '%s' found within the same import file (skipped)", idx+1, name, sku))
			continue
		}
		if cleanSKULower != "" {
			seenBatchSKUs[cleanSKULower] = true
		}

		// Resolve category
		catID := defaultCategoryID
		if item.CategoryID != "" {
			catID = item.CategoryID
		} else if item.CategoryName != "" {
			catID = resolveCategory(item.CategoryName)
		}
		if catID == "" {
			catID = defaultCategoryID
		}

		// Check if product exists in database
		var existingProductID string
		if cleanSKULower != "" {
			if pID, found := existingSKUs[cleanSKULower]; found {
				existingProductID = pID
			}
		}
		if existingProductID == "" {
			if pID, found := existingNames[cleanNameLower]; found {
				existingProductID = pID
			}
		}

		stock := item.StockQuantity
		if stock < 0 {
			stock = 0
		}
		minStock := item.MinStock
		if minStock <= 0 {
			minStock = 5
		}

		weight := 0.0

		// Prepare images and attributes JSONB
		imagesJSON := "[]"
		if item.ImageURL != "" {
			b, _ := json.Marshal([]string{item.ImageURL})
			imagesJSON = string(b)
		}

		attrs := make(map[string]interface{})
		if item.Barcode != "" {
			attrs["barcode"] = item.Barcode
		} else if sku != "" {
			attrs["barcode"] = sku
		}
		if item.Brand != "" {
			attrs["brand"] = item.Brand
		}
		if item.Unit != "" {
			attrs["unit"] = item.Unit
		}
		if item.HSNCode != "" {
			attrs["hsn_code"] = item.HSNCode
		}
		if item.GSTRate > 0 {
			attrs["gst_rate"] = item.GSTRate
		}
		attrsJSON := "{}"
		if len(attrs) > 0 {
			b, _ := json.Marshal(attrs)
			attrsJSON = string(b)
		}

		if existingProductID != "" {
			if updateExisting {
				toUpdate = append(toUpdate, sanitizedItem{
					rowNumber:    idx + 1,
					name:         name,
					description:  strings.TrimSpace(item.Description),
					sku:          sku,
					price:        item.Price,
					costPrice:    item.CostPrice,
					comparePrice: item.ComparePrice,
					categoryID:   catID,
					stock:        stock,
					minStock:     minStock,
					weight:       weight,
					existingID:   existingProductID,
					imagesJSON:   imagesJSON,
					attrsJSON:    attrsJSON,
				})
			} else {
				skippedCount++
				addErrMsg(fmt.Sprintf("Row %d ('%s'): Product with SKU '%s' already exists (skipped)", idx+1, name, sku))
			}
			continue
		}

		// Prepare new product slug & SKU
		if sku == "" {
			sku = fmt.Sprintf("SKU-%d-%04d", time.Now().Unix()%100000, rGenerator.Intn(10000))
			cleanSKULower = strings.ToLower(sku)
			seenBatchSKUs[cleanSKULower] = true
		}

		slug := reuse.Slugify(name)
		if slug == "" {
			slug = fmt.Sprintf("prod-%d", rGenerator.Intn(900000)+10000)
		}
		if len(slug) > 180 {
			slug = slug[:180]
		}
		slug = fmt.Sprintf("%s-%d", slug, rGenerator.Intn(900000)+10000)

		id := generateUUIDv4()

		toInsert = append(toInsert, sanitizedItem{
			id:           id,
			rowNumber:    idx + 1,
			name:         name,
			slug:         slug,
			description:  strings.TrimSpace(item.Description),
			sku:          sku,
			price:        item.Price,
			costPrice:    item.CostPrice,
			comparePrice: item.ComparePrice,
			categoryID:   catID,
			stock:        stock,
			minStock:     minStock,
			weight:       weight,
			imagesJSON:   imagesJSON,
			attrsJSON:    attrsJSON,
		})
	}

	const batchChunkSize = 250
	var importedCount int
	var updatedCount int

	// 4. Batch Updates (High-Speed Multi-Row Chunked Micro-Batches of 1,000 items)
	for i := 0; i < len(toUpdate); i += batchChunkSize {
		end := i + batchChunkSize
		if end > len(toUpdate) {
			end = len(toUpdate)
		}
		chunk := toUpdate[i:end]

		chunkTx, txErr := r.db.Begin(ctx)
		if txErr != nil {
			skippedCount += len(chunk)
			addErrMsg(fmt.Sprintf("Failed to begin transaction for update chunk: %v", txErr))
			continue
		}

		var updateValues []string
		var updateArgs []interface{}
		argIdx := 1

		var invValues []string
		var invArgs []interface{}
		invArgIdx := 1

		for _, it := range chunk {
			updateValues = append(updateValues, fmt.Sprintf(
				"($%d::uuid, $%d::numeric, $%d::numeric, $%d::numeric, $%d::uuid, $%d::text)",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5,
			))
			updateArgs = append(updateArgs, it.existingID, it.price, it.costPrice, it.comparePrice, it.categoryID, it.description)
			argIdx += 6

			invValues = append(invValues, fmt.Sprintf("($%d::uuid, $%d::int, 0, $%d::int, NOW())", invArgIdx, invArgIdx+1, invArgIdx+2))
			invArgs = append(invArgs, it.existingID, it.stock, it.minStock)
			invArgIdx += 3
		}

		updateArgs = append(updateArgs, shopID)
		shopIDParam := fmt.Sprintf("$%d", argIdx)

		bulkUpdateSQL := fmt.Sprintf(`
			UPDATE products AS p
			SET price = v.price,
			    cost_price = CASE WHEN v.cost_price > 0 THEN v.cost_price ELSE p.cost_price END,
			    compare_price = CASE WHEN v.compare_price > 0 THEN v.compare_price ELSE p.compare_price END,
			    category_id = COALESCE(v.category_id, p.category_id),
			    description = CASE WHEN v.description <> '' THEN v.description ELSE p.description END,
			    updated_at = NOW()
			FROM (VALUES %s) AS v(id, price, cost_price, compare_price, category_id, description)
			WHERE p.id = v.id AND p.shop_id = %s
		`, strings.Join(updateValues, ", "), shopIDParam)

		_, updateErr := chunkTx.Exec(ctx, bulkUpdateSQL, updateArgs...)
		if updateErr != nil {
			_ = chunkTx.Rollback(ctx)
			// Fallback to row-by-row on r.db for this chunk
			for _, it := range chunk {
				updateQuery := `
					UPDATE products
					SET price = $1,
					    cost_price = CASE WHEN $2 > 0 THEN $2 ELSE cost_price END,
					    compare_price = CASE WHEN $3 > 0 THEN $3 ELSE compare_price END,
					    category_id = COALESCE($4, category_id),
					    description = CASE WHEN $5 <> '' THEN $5 ELSE description END,
					    updated_at = NOW()
					WHERE id = $6 AND shop_id = $7
				`
				_, sErr := r.db.Exec(ctx, updateQuery, it.price, it.costPrice, it.comparePrice, it.categoryID, it.description, it.existingID, shopID)
				if sErr != nil {
					skippedCount++
					addErrMsg(fmt.Sprintf("Row %d ('%s'): Update failed: %v", it.rowNumber, it.name, sErr))
					continue
				}
				invQuery := `
					INSERT INTO inventory (product_id, quantity, reserved_quantity, low_stock_threshold, updated_at)
					VALUES ($1, $2, 0, $3, NOW())
					ON CONFLICT (product_id) DO UPDATE
					SET quantity = EXCLUDED.quantity, low_stock_threshold = EXCLUDED.low_stock_threshold, updated_at = NOW()
				`
				_, _ = r.db.Exec(ctx, invQuery, it.existingID, it.stock, it.minStock)
				updatedCount++
			}
			continue
		}

		// Update inventory in bulk for this chunk
		if len(invValues) > 0 {
			bulkInvSQL := fmt.Sprintf(`
				INSERT INTO inventory (product_id, quantity, reserved_quantity, low_stock_threshold, updated_at)
				VALUES %s
				ON CONFLICT (product_id) DO UPDATE
				SET quantity = EXCLUDED.quantity, low_stock_threshold = EXCLUDED.low_stock_threshold, updated_at = NOW()
			`, strings.Join(invValues, ", "))
			_, _ = chunkTx.Exec(ctx, bulkInvSQL, invArgs...)
		}

		if err := chunkTx.Commit(ctx); err == nil {
			updatedCount += len(chunk)
		} else {
			_ = chunkTx.Rollback(ctx)
			skippedCount += len(chunk)
			addErrMsg(fmt.Sprintf("Failed to commit update batch: %v", err))
		}
	}

	// 5. Batch Inserts (High-Speed Multi-Row Chunked Micro-Batches of 1,000 items)
	for i := 0; i < len(toInsert); i += batchChunkSize {
		end := i + batchChunkSize
		if end > len(toInsert) {
			end = len(toInsert)
		}
		chunk := toInsert[i:end]

		chunkTx, txErr := r.db.Begin(ctx)
		if txErr != nil {
			skippedCount += len(chunk)
			addErrMsg(fmt.Sprintf("Failed to begin transaction for insert chunk: %v", txErr))
			continue
		}

		var valueStrings []string
		var valueArgs []interface{}
		argIdx := 1

		var invValueStrings []string
		var invValueArgs []interface{}
		invArgIdx := 1

		for _, it := range chunk {
			valueStrings = append(valueStrings, fmt.Sprintf(
				"($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d::jsonb, $%d, true, false, '{}'::text[], $%d::jsonb, NOW(), NOW())",
				argIdx, argIdx+1, argIdx+2, argIdx+3, argIdx+4, argIdx+5, argIdx+6, argIdx+7, argIdx+8, argIdx+9, argIdx+10, argIdx+11, argIdx+12,
			))
			valueArgs = append(valueArgs,
				it.id,
				shopID,
				it.name,
				it.slug,
				it.description,
				it.sku,
				it.price,
				it.costPrice,
				it.comparePrice,
				it.categoryID,
				it.imagesJSON,
				it.weight,
				it.attrsJSON,
			)
			argIdx += 13

			invValueStrings = append(invValueStrings, fmt.Sprintf("($%d, $%d, 0, $%d, NOW())", invArgIdx, invArgIdx+1, invArgIdx+2))
			invValueArgs = append(invValueArgs, it.id, it.stock, it.minStock)
			invArgIdx += 3
		}

		insertSQL := fmt.Sprintf(`
			INSERT INTO products (
				id, shop_id, name, slug, description, sku, price, cost_price, compare_price,
				category_id, images, weight, is_active, is_featured, tags, attributes, created_at, updated_at
			) VALUES %s
		`, strings.Join(valueStrings, ", "))

		_, pErr := chunkTx.Exec(ctx, insertSQL, valueArgs...)
		if pErr != nil {
			_ = chunkTx.Rollback(ctx)
			// Fallback: If multi-row chunk encountered unexpected conflict, fallback to row-by-row on r.db
			for _, it := range chunk {
				singleInsert := `
					INSERT INTO products (
						id, shop_id, name, slug, description, sku, price, cost_price, compare_price,
						category_id, images, weight, is_active, is_featured, tags, attributes, created_at, updated_at
					) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, $12, true, false, '{}'::text[], $13::jsonb, NOW(), NOW())
				`
				_, sErr := r.db.Exec(ctx, singleInsert, it.id, shopID, it.name, it.slug, it.description, it.sku, it.price, it.costPrice, it.comparePrice, it.categoryID, it.imagesJSON, it.weight, it.attrsJSON)
				if sErr != nil {
					skippedCount++
					addErrMsg(fmt.Sprintf("Row %d ('%s'): Insert failed: %v", it.rowNumber, it.name, sErr))
					continue
				}
				invQuery := `
					INSERT INTO inventory (product_id, quantity, reserved_quantity, low_stock_threshold, updated_at)
					VALUES ($1, $2, 0, $3, NOW())
					ON CONFLICT (product_id) DO UPDATE
					SET quantity = EXCLUDED.quantity, low_stock_threshold = EXCLUDED.low_stock_threshold, updated_at = NOW()
				`
				_, _ = r.db.Exec(ctx, invQuery, it.id, it.stock, it.minStock)
				importedCount++
			}
			continue
		}

		// Insert inventory in bulk for this chunk
		if len(invValueStrings) > 0 {
			invBulkSQL := fmt.Sprintf(`
				INSERT INTO inventory (product_id, quantity, reserved_quantity, low_stock_threshold, updated_at)
				VALUES %s
				ON CONFLICT (product_id) DO UPDATE
				SET quantity = EXCLUDED.quantity, low_stock_threshold = EXCLUDED.low_stock_threshold, updated_at = NOW()
			`, strings.Join(invValueStrings, ", "))

			_, _ = chunkTx.Exec(ctx, invBulkSQL, invValueArgs...)
		}

		if err := chunkTx.Commit(ctx); err == nil {
			importedCount += len(chunk)
		} else {
			_ = chunkTx.Rollback(ctx)
			skippedCount += len(chunk)
			addErrMsg(fmt.Sprintf("Failed to commit insert batch: %v", err))
		}
	}

	return &dto.BulkImportResponse{
		TotalRows:     len(items),
		ImportedCount: importedCount,
		UpdatedCount:  updatedCount,
		SkippedCount:  skippedCount,
		Errors:        errMsgs,
	}, nil
}

func isDigitsOnly(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

