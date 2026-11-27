// Package repository handle db query.
package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"shopMe/internal/handler/dto"
	"shopMe/internal/handler/model"
	"shopMe/internal/utils"
	"sort"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"go.uber.org/zap"
)

func (r *ProductRepo) AdjustInventoryStock(ctx context.Context, shopID, productID string, adjustment int, threshold *int) (*model.Inventory, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM products WHERE id = $1 AND shop_id = $2)`, productID, shopID).Scan(&exists)
	if err != nil || !exists {
		return nil, ErrProductNotFound
	}

	query := `
		UPDATE inventory
		SET quantity = GREATEST(0, quantity + $1),
		    low_stock_threshold = COALESCE($2, low_stock_threshold),
		    updated_at = NOW()
		WHERE product_id = $3
		RETURNING product_id, quantity, reserved_quantity, low_stock_threshold
	`
	inv := &model.Inventory{}
	err = r.db.QueryRow(ctx, query, adjustment, threshold, productID).Scan(
		&inv.ProductID,
		&inv.Quantity,
		&inv.ReservedQuantity,
		&inv.LowStockThreshold,
	)
	if err != nil {
		r.logger.Error("failed to adjust inventory", zap.Error(err), zap.String("product_id", productID))
		return nil, err
	}
	inv.AvailableQuantity = inv.Quantity - inv.ReservedQuantity
	if inv.AvailableQuantity < 0 {
		inv.AvailableQuantity = 0
	}
	return inv, nil
}

func (r *ProductRepo) GetLowStockProducts(ctx context.Context, shopID string) ([]*dto.LowStockProduct, error) {
	query := `
		SELECT p.id, p.name, p.sku, p.price,
		       i.quantity, i.reserved_quantity, i.low_stock_threshold
		FROM products p
		JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 AND p.is_active = true AND i.quantity <= i.low_stock_threshold
		ORDER BY i.quantity ASC, p.name ASC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to get low stock products", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var items []*dto.LowStockProduct
	for rows.Next() {
		item := &dto.LowStockProduct{}
		err := rows.Scan(
			&item.ProductID,
			&item.Name,
			&item.SKU,
			&item.Price,
			&item.CurrentStock,
			&item.ReservedStock,
			&item.LowStockThreshold,
		)
		if err != nil {
			r.logger.Error("failed to scan low stock row", zap.Error(err))
			return nil, err
		}

		avail := item.CurrentStock - item.ReservedStock
		if avail < 0 {
			avail = 0
		}
		item.AvailableStock = avail

		suggested := (item.LowStockThreshold * 3) - item.CurrentStock
		if suggested < 10 {
			suggested = 10
		}
		item.SuggestedReorderQty = suggested

		items = append(items, item)
	}

	return items, nil
}

func (r *ProductRepo) DismissStockAlert(ctx context.Context, shopID, productID string) error {
	query := `
		UPDATE inventory i
		SET low_stock_threshold = -1, updated_at = NOW()
		FROM products p
		WHERE i.product_id = p.id AND p.id = $1 AND p.shop_id = $2
	`
	_, err := r.db.Exec(ctx, query, productID, shopID)
	return err
}

// GetMonthlyProfitAnalytics aggregates sales, revenue, cost and net profit for a shop in a specific month (combining POS bills and completed reservations).
func (r *ProductRepo) GetMonthlyProfitAnalytics(ctx context.Context, shopID string, year, month int) (*dto.MonthlyProfitResponse, error) {
	query := `
		WITH pos_sales AS (
			SELECT
				COUNT(b.id) AS sales_count,
				COALESCE(SUM(b.total_amount), 0.0) AS revenue,
				COALESCE(SUM(b.total_cost), 0.0) AS cost,
				COALESCE(SUM(items.qty), 0) AS items_count
			FROM pos_bills b
			LEFT JOIN (
				SELECT bill_id, SUM(quantity) AS qty
				FROM pos_bill_items
				GROUP BY bill_id
			) items ON items.bill_id = b.id
			WHERE b.shop_id = $1
			  AND COALESCE(b.status, '') <> 'cancelled'
			  AND EXTRACT(YEAR FROM b.created_at) = $2
			  AND EXTRACT(MONTH FROM b.created_at) = $3
		),
		res_sales AS (
			SELECT 
				COUNT(r.id) AS sales_count,
				COALESCE(SUM(r.quantity), 0) AS items_count,
				COALESCE(SUM(r.quantity * p.price), 0.0) AS revenue,
				COALESCE(SUM(r.quantity * COALESCE(p.cost_price, 0)), 0.0) AS cost
			FROM reservations r
			JOIN products p ON r.product_id = p.id
			WHERE r.shop_id = $1
			  AND r.status = 'completed'
			  AND EXTRACT(YEAR FROM r.completed_at) = $2
			  AND EXTRACT(MONTH FROM r.completed_at) = $3
		)
		SELECT
			COALESCE(p.sales_count, 0) + COALESCE(r.sales_count, 0),
			COALESCE(p.items_count, 0) + COALESCE(r.items_count, 0),
			COALESCE(p.revenue, 0.0) + COALESCE(r.revenue, 0.0),
			COALESCE(p.cost, 0.0) + COALESCE(r.cost, 0.0)
		FROM pos_sales p
		CROSS JOIN res_sales r;
	`
	var totalSales, totalItems int
	var totalRevenue, totalCost float64
	err := r.db.QueryRow(ctx, query, shopID, year, month).Scan(&totalSales, &totalItems, &totalRevenue, &totalCost)
	if err != nil {
		r.logger.Error("failed to get monthly profit analytics", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}

	netProfit := totalRevenue - totalCost
	var avgMargin float64
	if totalRevenue > 0 {
		avgMargin = math.Round((netProfit/totalRevenue)*1000) / 10
	}

	monthName := time.Month(month).String()

	return &dto.MonthlyProfitResponse{
		Month:               monthName,
		Year:                year,
		TotalCompletedSales: totalSales,
		TotalItemsSold:      totalItems,
		TotalRevenue:        totalRevenue,
		TotalCost:           totalCost,
		NetProfit:           netProfit,
		AverageMarginPct:    avgMargin,
	}, nil
}

// GetProductPerformanceMatrix categorizes products into Best, Worst, Old/Dead Stock, and New Arrivals.
func (r *ProductRepo) GetProductPerformanceMatrix(ctx context.Context, shopID string) (*dto.ProductMatrixResponse, error) {
	query := `
		WITH pos_sales AS (
			SELECT
				i.product_id,
				SUM(i.quantity)::int AS sold_qty,
				SUM(
					CASE
						WHEN COALESCE(i.unit_cost, 0) > 0 THEN
							COALESCE(i.total_price, i.unit_price * i.quantity)
							- CASE
								WHEN COALESCE(b.subtotal, 0) > 0
									THEN COALESCE(b.discount_amount, 0) * COALESCE(i.total_price, i.unit_price * i.quantity) / b.subtotal
								ELSE 0
							END
							- i.unit_cost * i.quantity
						ELSE 0
					END
				) AS total_profit
			FROM pos_bill_items i
			JOIN pos_bills b ON b.id = i.bill_id
			WHERE b.shop_id = $1 AND COALESCE(b.status, '') <> 'cancelled'
			GROUP BY i.product_id
		),
		reservation_sales AS (
			SELECT
				r.product_id,
				SUM(r.quantity)::int AS sold_qty,
				SUM(
					CASE
						WHEN COALESCE(p.cost_price, 0) > 0
							THEN r.quantity * (p.price - p.cost_price)
						ELSE 0
					END
				) AS total_profit
			FROM reservations r
			JOIN products p ON p.id = r.product_id
			WHERE r.status = 'completed' AND r.shop_id = $1
			GROUP BY r.product_id
		),
		sold AS (
			SELECT product_id, SUM(sold_qty)::int AS sold_qty, SUM(total_profit) AS total_profit
			FROM (
				SELECT product_id, sold_qty, total_profit FROM pos_sales
				UNION ALL
				SELECT product_id, sold_qty, total_profit FROM reservation_sales
			) combined_sales
			GROUP BY product_id
		)
		SELECT 
			p.id, p.name, p.sku, p.price, COALESCE(p.cost_price, 0),
			COALESCE(i.quantity, 0),
			COALESCE(sold.sold_qty, 0) AS total_sold_qty,
			COALESCE(sold.total_profit, 0) AS total_profit,
			COALESCE(EXTRACT(DAY FROM (NOW() - p.created_at))::int, 0) AS days_in_stock
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		LEFT JOIN sold ON sold.product_id = p.id
		WHERE p.shop_id = $1 AND p.is_active = true
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to get product performance matrix", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var allItems []*dto.ProductPerformanceItem
	for rows.Next() {
		item := &dto.ProductPerformanceItem{}
		err := rows.Scan(
			&item.ProductID,
			&item.Name,
			&item.SKU,
			&item.Price,
			&item.CostPrice,
			&item.CurrentStock,
			&item.TotalSoldQty,
			&item.TotalProfit,
			&item.DaysInStock,
		)
		if err != nil {
			r.logger.Error("failed to scan performance item", zap.Error(err))
			return nil, err
		}

		if item.CostPrice > 0 {
			item.UnitProfit = item.Price - item.CostPrice
			item.ProfitMarginPct = math.Round((item.UnitProfit/item.Price)*1000) / 10
		}

		allItems = append(allItems, item)
	}

	matrix := &dto.ProductMatrixResponse{
		BestProfitable:  []*dto.ProductPerformanceItem{},
		WorstProfitable: []*dto.ProductPerformanceItem{},
		OldDeadStock:    []*dto.ProductPerformanceItem{},
		NewArrivals:     []*dto.ProductPerformanceItem{},
	}

	// 1. Sort by total profit desc for BestProfitable (top 5)
	bestList := make([]*dto.ProductPerformanceItem, len(allItems))
	copy(bestList, allItems)
	sort.Slice(bestList, func(i, j int) bool {
		return bestList[i].TotalProfit > bestList[j].TotalProfit
	})
	if len(bestList) > 5 {
		bestList = bestList[:5]
	}
	matrix.BestProfitable = bestList

	// 2. Sort by margin asc for WorstProfitable (bottom 5 with lowest margin)
	worstList := make([]*dto.ProductPerformanceItem, len(allItems))
	copy(worstList, allItems)
	sort.Slice(worstList, func(i, j int) bool {
		return worstList[i].ProfitMarginPct < worstList[j].ProfitMarginPct
	})
	if len(worstList) > 5 {
		worstList = worstList[:5]
	}
	matrix.WorstProfitable = worstList

	// 3. Old / Dead Stock (days_in_stock >= 60 and total_sold_qty <= 1)
	for _, it := range allItems {
		if it.DaysInStock >= 60 && it.TotalSoldQty <= 1 {
			matrix.OldDeadStock = append(matrix.OldDeadStock, it)
		}
		if it.DaysInStock <= 30 {
			matrix.NewArrivals = append(matrix.NewArrivals, it)
		}
	}

	return matrix, nil
}

// FindNearbyProducts searches for in-stock products in shops within radiusKm from (lat, lng), sorted by distance.
func (r *ProductRepo) FindNearbyProducts(
	ctx context.Context,
	lat, lng, radiusKm float64,
	query, category string,
	openNowOnly bool,
	page, limit int,
) ([]*dto.NearbyProductItem, int, error) {
	cleanedNearbyQ := utils.CleanSearchQuery(query)
	nearbyCacheKey := ""
	if cleanedNearbyQ != "" {
		nearbyCacheKey = fmt.Sprintf("p_nrby:%s:lat:%.3f:lng:%.3f:rad:%.1f:cat:%s:open:%t:p:%d:l:%d",
			cleanedNearbyQ, lat, lng, radiusKm, category, openNowOnly, page, limit)
		if cachedItems, cachedCount, found := r.getCachedNearbySearch(nearbyCacheKey); found {
			return cachedItems, cachedCount, nil
		}
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}
	offset := (page - 1) * limit

	var whereClauses []string
	args := []interface{}{lat, lng, radiusKm}
	argIdx := 4

	whereClauses = append(whereClauses, "p.is_active = true")
	whereClauses = append(whereClauses, "s.is_active = true")
	whereClauses = append(whereClauses, "s.status = 'active'")
	whereClauses = append(whereClauses, "s.latitude IS NOT NULL AND s.longitude IS NOT NULL")
	whereClauses = append(whereClauses, "(i.quantity - i.reserved_quantity) > 0")

	radiusMeters := radiusKm * 1000.0
	args[2] = radiusMeters // Replace radiusKm with radiusMeters for ST_DWithin

	// PostGIS Level 1 Spatial Architecture:
	// $1 = lat, $2 = lng, $3 = radiusMeters. ST_MakePoint takes (longitude, latitude)
	userPoint := "ST_SetSRID(ST_MakePoint($2, $1), 4326)::geography"
	distanceFormula := fmt.Sprintf("ROUND((ST_Distance(s.location, %s) / 1000.0)::numeric, 2)::double precision", userPoint)

	whereClauses = append(whereClauses,
		"s.location IS NOT NULL",
		fmt.Sprintf("ST_DWithin(s.location, %s, $3)", userPoint),
	)

	if openNowOnly {
		whereClauses = append(whereClauses, "s.is_open = true")
	}

	trimmedQ := strings.TrimSpace(query)
	if trimmedQ != "" {
		groups := utils.ExpandHinglishSearchGroups(trimmedQ)
		if len(groups) > 0 {
			for _, group := range groups {
				var groupOrs []string
				for _, term := range group.Terms {
					cleanTerm := strings.TrimSpace(term)
					if cleanTerm == "" {
						continue
					}
					wild := "%" + cleanTerm + "%"
					if len(cleanTerm) >= 3 {
						groupOrs = append(groupOrs, fmt.Sprintf("(p.name ILIKE $%d OR word_similarity($%d, p.name) >= 0.35 OR p.name %% $%d OR COALESCE(p.description, '') ILIKE $%d OR COALESCE(p.sku, '') ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx+1, argIdx+1, argIdx, argIdx, argIdx, argIdx))
						args = append(args, wild, cleanTerm)
						argIdx += 2
					} else {
						groupOrs = append(groupOrs, fmt.Sprintf("(p.name ILIKE $%d OR COALESCE(p.description, '') ILIKE $%d OR COALESCE(p.sku, '') ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx))
						args = append(args, wild)
						argIdx++
					}
				}
				if len(groupOrs) > 0 {
					whereClauses = append(whereClauses, "("+strings.Join(groupOrs, " OR ")+")")
				}
			}
		} else {
			cleanTerm := trimmedQ
			wild := "%" + cleanTerm + "%"
			if len(cleanTerm) >= 3 {
				whereClauses = append(whereClauses, fmt.Sprintf("(p.name ILIKE $%d OR word_similarity($%d, p.name) >= 0.35 OR p.name %% $%d OR COALESCE(p.description, '') ILIKE $%d OR COALESCE(p.sku, '') ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx+1, argIdx+1, argIdx, argIdx, argIdx, argIdx))
				args = append(args, wild, cleanTerm)
				argIdx += 2
			} else {
				whereClauses = append(whereClauses, fmt.Sprintf("(p.name ILIKE $%d OR COALESCE(p.description, '') ILIKE $%d OR COALESCE(p.sku, '') ILIKE $%d OR immutable_array_to_string(p.tags, ' ') ILIKE $%d OR COALESCE(p.attributes::text, '') ILIKE $%d)", argIdx, argIdx, argIdx, argIdx, argIdx))
				args = append(args, wild)
				argIdx++
			}
		}
	}

	trimmedCat := strings.TrimSpace(category)
	if trimmedCat != "" {
		var catOrs []string
		catOrs = append(catOrs, fmt.Sprintf("s.category ILIKE $%d", argIdx))
		args = append(args, "%"+trimmedCat+"%")
		argIdx++

		synonymSlugs := utils.GetCategorySynonymSlugs(trimmedCat)
		synonymNames := utils.GetCategorySynonymNames(trimmedCat)

		if len(synonymSlugs) > 0 {
			catOrs = append(catOrs, fmt.Sprintf("p.category_id IN (SELECT id FROM categories WHERE slug = ANY($%d))", argIdx))
			args = append(args, synonymSlugs)
			argIdx++

			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category_name', '')) = ANY($%d)", argIdx))
			args = append(args, synonymSlugs)
			argIdx++

			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category', '')) = ANY($%d)", argIdx))
			args = append(args, synonymSlugs)
			argIdx++
		}

		if len(synonymNames) > 0 {
			var lowerNames []string
			for _, n := range synonymNames {
				lowerNames = append(lowerNames, strings.ToLower(n))
			}
			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category_name', '')) = ANY($%d)", argIdx))
			args = append(args, lowerNames)
			argIdx++

			catOrs = append(catOrs, fmt.Sprintf("LOWER(COALESCE(p.attributes->>'category', '')) = ANY($%d)", argIdx))
			args = append(args, lowerNames)
			argIdx++
		}

		whereClauses = append(whereClauses, "("+strings.Join(catOrs, " OR ")+")")
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM products p
		JOIN shops s ON p.shop_id = s.id
		JOIN inventory i ON p.id = i.product_id
		WHERE %s
	`, whereSQL)

	var totalCount int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&totalCount); err != nil {
		r.logger.Error("failed to count nearby products", zap.Error(err))
		return nil, 0, err
	}

	if totalCount == 0 {
		return []*dto.NearbyProductItem{}, 0, nil
	}

	orderBy := "distance_km ASC, p.price ASC"
	if trimmedQ != "" && len(trimmedQ) >= 3 {
		orderBy = fmt.Sprintf("GREATEST(word_similarity($%d, p.name), similarity(p.name, $%d)) DESC, distance_km ASC, p.price ASC", argIdx, argIdx)
		args = append(args, trimmedQ)
		argIdx++
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			p.id, p.name, p.slug, p.price, COALESCE(p.compare_price, 0), COALESCE(p.sku, ''),
			COALESCE(p.is_price_public, true),
			COALESCE(p.images, '[]'::jsonb), (i.quantity - i.reserved_quantity) AS avail_qty,
			s.id AS shop_id, s.name AS shop_name, s.slug AS shop_slug, s.address AS shop_address, s.phone AS shop_phone,
			COALESCE(s.city, ''), COALESCE(s.latitude, 0), COALESCE(s.longitude, 0),
			s.is_open, COALESCE(s.opening_time, ''), COALESCE(s.closing_time, ''), COALESCE(s.weekly_off, ''),
			COALESCE(p.category_id::text, ''), COALESCE(c.name, ''), COALESCE(c.slug, ''), COALESCE(p.attributes, '{}'::jsonb),
			%s AS distance_km
		FROM products p
		JOIN shops s ON p.shop_id = s.id
		JOIN inventory i ON p.id = i.product_id
		LEFT JOIN categories c ON p.category_id = c.id
		WHERE %s
		ORDER BY %s
		LIMIT $%d OFFSET $%d
	`, distanceFormula, whereSQL, orderBy, argIdx, argIdx+1)

	args = append(args, limit, offset)

	rows, err := r.db.Query(ctx, sqlQuery, args...)
	if err != nil {
		r.logger.Error("failed to find nearby products", zap.Error(err))
		return nil, 0, err
	}
	defer rows.Close()

	var items []*dto.NearbyProductItem

	for rows.Next() {
		item := &dto.NearbyProductItem{}
		var imagesBytes []byte
		var openTime, closeTime, weeklyOff string

		var catID, catName, catSlug string
		var attrsBytes []byte
		err := rows.Scan(
			&item.ProductID,
			&item.Name,
			&item.Slug,
			&item.Price,
			&item.ComparePrice,
			&item.SKU,
			&item.IsPricePublic,
			&imagesBytes,
			&item.AvailableQuantity,
			&item.ShopID,
			&item.ShopName,
			&item.ShopSlug,
			&item.ShopAddress,
			&item.ShopPhone,
			&item.ShopCity,
			&item.ShopLatitude,
			&item.ShopLongitude,
			&item.IsOpen,
			&openTime,
			&closeTime,
			&weeklyOff,
			&catID,
			&catName,
			&catSlug,
			&attrsBytes,
			&item.DistanceKm,
		)
		item.ID = item.ProductID
		item.IsInStock = item.AvailableQuantity > 0
		if err != nil {
			r.logger.Error("failed to scan nearby product row", zap.Error(err))
			return nil, 0, err
		}

		item.Images = []string{}
		item.CategoryID = catID
		item.CategoryName = catName
		item.CategorySlug = catSlug
		if item.CategoryName == "" && len(attrsBytes) > 0 {
			var attrs map[string]interface{}
			if err := json.Unmarshal(attrsBytes, &attrs); err == nil {
				if c, ok := attrs["category_name"].(string); ok && c != "" {
					item.CategoryName = c
				} else if c, ok := attrs["category"].(string); ok && c != "" {
					item.CategoryName = c
				}
			}
		}

		if len(imagesBytes) > 0 {
			_ = json.Unmarshal(imagesBytes, &item.Images)
		}

		dummyShop := &model.Shop{
			IsOpen:      item.IsOpen,
			IsActive:    true,
			OpeningTime: openTime,
			ClosingTime: closeTime,
			WeeklyOff:   weeklyOff,
		}
		computeShopOpenStatus(dummyShop)
		item.IsCurrentlyOpen = dummyShop.IsCurrentlyOpen

		item.DistanceKm = math.Round(item.DistanceKm*100) / 100
		items = append(items, item)
	}

	if nearbyCacheKey != "" && len(items) > 0 {
		r.setCachedNearbySearch(nearbyCacheKey, items, totalCount)
	}

	return items, totalCount, nil
}

// SuggestProducts returns top matching product names for fast autocomplete search.
// Leverages Go in-memory LRU prefix cache (<0.5ms) with fallback to GIN trigram index and Hinglish expansion.
func (r *ProductRepo) SuggestProducts(ctx context.Context, rawQuery string, shopID string, limit int) ([]string, error) {
	clean := utils.CleanSearchQuery(rawQuery)
	if clean == "" {
		return []string{}, nil
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	// 1. In-Memory LRU Prefix Cache Lookup (SLA <0.5ms, 90% DB query reduction)
	cacheKey := fmt.Sprintf("sugg:s:%s:q:%s:l:%d", shopID, clean, limit)
	if cached, found := r.getCachedSuggestions(cacheKey); found {
		return cached, nil
	}

	// 2. Expand Hinglish colloquial synonyms (e.g. "dud" -> "milk", "tel" -> "oil")
	hinglishTerms := []string{clean}
	groups := utils.ExpandHinglishSearchGroups(clean)
	for _, g := range groups {
		for _, t := range g.Terms {
			t = strings.TrimSpace(t)
			if t != clean && len(t) >= 3 {
				hinglishTerms = append(hinglishTerms, t)
			}
		}
	}

	var sqlQuery string
	var rows pgx.Rows
	var err error

	wild := "%" + clean + "%"
	if len(clean) >= 3 {
		if shopID != "" {
			sqlQuery = `
				SELECT name FROM (
					SELECT p.name,
					       CASE WHEN p.name ILIKE ($3 || '%') THEN 0 
					            WHEN p.name ILIKE ('% ' || $3 || '%') THEN 1 
					            ELSE 2 
					       END AS prefix_rank,
					       GREATEST(word_similarity($3, p.name), similarity(p.name, $3)) AS sim
					FROM products p
					WHERE p.is_active = true
					  AND p.shop_id = $1
					  AND (p.name ILIKE $2 OR word_similarity($3, p.name) >= 0.28 OR p.name % $3 OR (cardinality($5::text[]) > 1 AND p.name ILIKE ANY($5)))
					GROUP BY p.name
				) sub
				ORDER BY prefix_rank ASC, sim DESC, name ASC
				LIMIT $4;
			`
			var wildHinglish []string
			for _, ht := range hinglishTerms {
				wildHinglish = append(wildHinglish, "%"+ht+"%")
			}
			rows, err = r.db.Query(ctx, sqlQuery, shopID, wild, clean, limit, wildHinglish)
		} else {
			sqlQuery = `
				SELECT name FROM (
					SELECT p.name,
					       CASE WHEN p.name ILIKE ($2 || '%') THEN 0 
					            WHEN p.name ILIKE ('% ' || $2 || '%') THEN 1 
					            ELSE 2 
					       END AS prefix_rank,
					       GREATEST(word_similarity($2, p.name), similarity(p.name, $2)) AS sim
					FROM products p
					WHERE p.is_active = true
					  AND (p.name ILIKE $1 OR word_similarity($2, p.name) >= 0.28 OR p.name % $2 OR (cardinality($4::text[]) > 1 AND p.name ILIKE ANY($4)))
					GROUP BY p.name
				) sub
				ORDER BY prefix_rank ASC, sim DESC, name ASC
				LIMIT $3;
			`
			var wildHinglish []string
			for _, ht := range hinglishTerms {
				wildHinglish = append(wildHinglish, "%"+ht+"%")
			}
			rows, err = r.db.Query(ctx, sqlQuery, wild, clean, limit, wildHinglish)
		}
	} else {
		if shopID != "" {
			sqlQuery = `
				SELECT name FROM (
					SELECT p.name,
					       CASE WHEN p.name ILIKE ($3 || '%') THEN 0 ELSE 1 END AS prefix_rank
					FROM products p
					WHERE p.is_active = true
					  AND p.shop_id = $1
					  AND p.name ILIKE $2
					GROUP BY p.name
				) sub
				ORDER BY prefix_rank ASC, name ASC
				LIMIT $4;
			`
			rows, err = r.db.Query(ctx, sqlQuery, shopID, wild, clean, limit)
		} else {
			sqlQuery = `
				SELECT name FROM (
					SELECT p.name,
					       CASE WHEN p.name ILIKE ($2 || '%') THEN 0 ELSE 1 END AS prefix_rank
					FROM products p
					WHERE p.is_active = true
					  AND p.name ILIKE $1
					GROUP BY p.name
				) sub
				ORDER BY prefix_rank ASC, name ASC
				LIMIT $3;
			`
			rows, err = r.db.Query(ctx, sqlQuery, wild, clean, limit)
		}
	}

	if err != nil {
		r.logger.Error("failed to suggest products", zap.Error(err))
		return nil, err
	}
	defer rows.Close()

	var suggestions []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			suggestions = append(suggestions, name)
		}
	}
	if err := rows.Err(); err != nil {
		r.logger.Error("rows iteration error in suggest products", zap.Error(err))
		return nil, err
	}
	if suggestions == nil {
		suggestions = []string{}
	}

	// Store in in-memory LRU cache
	r.setCachedSuggestions(cacheKey, suggestions)
	return suggestions, nil
}

// CreateStockAlert registers a customer's request to be alerted when an out-of-stock product is restocked.
func (r *ProductRepo) CreateStockAlert(ctx context.Context, productID, shopID, phone, name string) error {
	query := `
		INSERT INTO product_stock_alerts (product_id, shop_id, customer_phone, customer_name, notified, created_at)
		VALUES ($1, $2, $3, $4, false, NOW())
		ON CONFLICT (product_id, customer_phone) DO UPDATE 
		SET customer_name = EXCLUDED.customer_name, notified = false, created_at = NOW()
	`
	_, err := r.db.Exec(ctx, query, productID, shopID, strings.TrimSpace(phone), strings.TrimSpace(name))
	if err != nil {
		r.logger.Error("failed to create product stock alert", zap.Error(err), zap.String("product_id", productID))
		return err
	}
	return nil
}

// GetDemandWatchlist aggregates unnotified customer interest for out-of-stock and low-stock items.
func (r *ProductRepo) GetDemandWatchlist(ctx context.Context, shopID string) ([]*dto.DemandWatchlistItem, error) {
	query := `
		SELECT 
			p.id, p.name, p.sku, 
			(i.quantity - i.reserved_quantity) AS current_stock,
			COUNT(a.id) AS waiting_customers_count
		FROM product_stock_alerts a
		JOIN products p ON a.product_id = p.id
		JOIN inventory i ON p.id = i.product_id
		WHERE a.shop_id = $1 AND a.notified = false
		GROUP BY p.id, p.name, p.sku, i.quantity, i.reserved_quantity
		ORDER BY waiting_customers_count DESC, p.name ASC
	`
	rows, err := r.db.Query(ctx, query, shopID)
	if err != nil {
		r.logger.Error("failed to query demand watchlist", zap.Error(err), zap.String("shop_id", shopID))
		return nil, err
	}
	defer rows.Close()

	var list []*dto.DemandWatchlistItem
	for rows.Next() {
		var item dto.DemandWatchlistItem
		if err := rows.Scan(&item.ProductID, &item.ProductName, &item.SKU, &item.CurrentStock, &item.WaitingCustomersCount); err != nil {
			return nil, err
		}
		list = append(list, &item)
	}
	if list == nil {
		list = []*dto.DemandWatchlistItem{}
	}
	return list, nil
}

// CountWaitingCustomers returns the number of customers awaiting a restock of the product.
func (r *ProductRepo) CountWaitingCustomers(ctx context.Context, productID string) (int, error) {
	var count int
	err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM product_stock_alerts WHERE product_id = $1 AND notified = false`, productID).Scan(&count)
	return count, err
}

// FindActiveProductsByTokens searches active store products by a set of keyword tokens for Parchi parsing.
func (r *ProductRepo) FindActiveProductsByTokens(ctx context.Context, shopID string, tokens []string, limit int) ([]*model.Product, error) {
	if len(tokens) == 0 {
		return []*model.Product{}, nil
	}
	if limit <= 0 || limit > 10 {
		limit = 5
	}

	conditions := make([]string, len(tokens))
	args := []interface{}{shopID}
	for i, t := range tokens {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(t))+"%")
		conditions[i] = fmt.Sprintf("(LOWER(p.name) LIKE $%d OR LOWER(COALESCE(p.sku, '')) LIKE $%d)", len(args), len(args))
	}

	query := fmt.Sprintf(`
		SELECT p.id, p.shop_id, p.name, p.sku, p.price, COALESCE(p.cost_price, 0),
		       COALESCE(i.quantity, 0) - COALESCE(i.reserved_quantity, 0) AS available_stock
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		WHERE p.shop_id = $1 AND p.is_active = true AND (%s)
		ORDER BY available_stock DESC, p.name ASC
		LIMIT %d
	`, strings.Join(conditions, " OR "), limit)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var products []*model.Product
	for rows.Next() {
		var p model.Product
		var availStock int
		if err := rows.Scan(&p.ID, &p.ShopID, &p.Name, &p.SKU, &p.Price, &p.CostPrice, &availStock); err != nil {
			return nil, err
		}
		p.Inventory = &model.Inventory{AvailableQuantity: availStock, Quantity: availStock}
		products = append(products, &p)
	}
	return products, nil
}

// GetDukaanPulseMetrics computes pre-aggregated headline sales, a 7-day trend series,
// and inventory/khata operational counts directly in PostgreSQL.
func (r *ProductRepo) GetDukaanPulseMetrics(ctx context.Context, shopID string, now time.Time) (*dto.PulseHeadline, []dto.PulseDayPoint, *dto.PulseOperations, error) {
	refDate := now.Format("2006-01-02")

	// 1. Headline sales rollup (Today, Yesterday, Current Month, Previous Month)
	headlineQuery := `
		SELECT
			COALESCE(SUM(CASE WHEN created_at::date = $2::date THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN created_at::date = $2::date THEN total_amount - total_cost ELSE 0 END), 0),
			COUNT(CASE WHEN created_at::date = $2::date THEN 1 END),
			COALESCE(SUM(CASE WHEN created_at::date = ($2::date - INTERVAL '1 day')::date THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN created_at >= date_trunc('month', $2::date) THEN total_amount ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN created_at >= date_trunc('month', $2::date) THEN total_amount - total_cost ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN created_at >= date_trunc('month', $2::date) - INTERVAL '1 month'
			                   AND created_at < date_trunc('month', $2::date) THEN total_amount ELSE 0 END), 0)
		FROM pos_bills
		WHERE shop_id = $1
		  AND COALESCE(status, '') <> 'cancelled'
		  AND created_at >= date_trunc('month', $2::date) - INTERVAL '1 month'
	`
	headline := &dto.PulseHeadline{}
	if err := r.db.QueryRow(ctx, headlineQuery, shopID, refDate).Scan(
		&headline.TodayRevenue,
		&headline.TodayProfit,
		&headline.TodayBills,
		&headline.YesterdayRevenue,
		&headline.MonthRevenue,
		&headline.MonthGrossProfit,
		&headline.PrevMonthRevenue,
	); err != nil {
		r.logger.Error("failed to query dukaan pulse headline", zap.Error(err), zap.String("shop_id", shopID))
		return nil, nil, nil, err
	}

	// 2. 7-day pre-bucketed chart series (guaranteed 7 days including 0-sale days)
	trendQuery := `
		WITH days AS (
			SELECT generate_series(($2::date - INTERVAL '6 days')::date, $2::date, INTERVAL '1 day')::date AS day
		)
		SELECT
			to_char(d.day, 'YYYY-MM-DD') AS date_str,
			to_char(d.day, 'Dy') AS day_label,
			COALESCE(SUM(b.total_amount), 0) AS revenue,
			COALESCE(SUM(b.total_amount - b.total_cost), 0) AS profit,
			COUNT(b.id) AS bills
		FROM days d
		LEFT JOIN pos_bills b
			ON b.shop_id = $1
		   AND COALESCE(b.status, '') <> 'cancelled'
		   AND b.created_at::date = d.day
		GROUP BY d.day
		ORDER BY d.day ASC
	`
	rows, err := r.db.Query(ctx, trendQuery, shopID, refDate)
	if err != nil {
		r.logger.Error("failed to query dukaan pulse 7-day trend", zap.Error(err), zap.String("shop_id", shopID))
		return nil, nil, nil, err
	}
	defer rows.Close()

	trend := make([]dto.PulseDayPoint, 0, 7)
	for rows.Next() {
		var pt dto.PulseDayPoint
		if err := rows.Scan(&pt.Date, &pt.Label, &pt.Revenue, &pt.Profit, &pt.Bills); err != nil {
			return nil, nil, nil, err
		}
		trend = append(trend, pt)
	}

	// 3. Operations summary (Inventory health + Khata outstanding)
	opsQuery := `
		WITH inv_stats AS (
			SELECT
				COUNT(CASE WHEN COALESCE(i.quantity, 0) <= 0 THEN 1 END) AS out_of_stock,
				COUNT(CASE WHEN COALESCE(i.quantity, 0) > 0 AND i.quantity <= i.low_stock_threshold THEN 1 END) AS low_stock,
				COUNT(CASE WHEN COALESCE(i.quantity, 0) > 0
				            AND p.created_at < ($2::date - INTERVAL '45 days')
				            AND NOT EXISTS (
				                SELECT 1
				                FROM pos_bill_items bi
				                JOIN pos_bills b ON b.id = bi.bill_id
				                WHERE bi.product_id = p.id
				                  AND b.shop_id = $1
				                  AND COALESCE(b.status, '') <> 'cancelled'
				                  AND b.created_at >= ($2::date - INTERVAL '45 days')
				            ) THEN 1 END) AS dead_stock
			FROM products p
			LEFT JOIN inventory i ON i.product_id = p.id
			WHERE p.shop_id = $1 AND p.is_active = true
		),
		khata_stats AS (
			SELECT
				COALESCE(SUM(CASE WHEN current_balance > 0 THEN current_balance ELSE 0 END), 0) AS outstanding,
				COUNT(CASE WHEN current_balance > 0 THEN 1 END) AS customers
			FROM customer_khata
			WHERE shop_id = $1
		)
		SELECT
			inv_stats.low_stock,
			inv_stats.out_of_stock,
			inv_stats.dead_stock,
			khata_stats.outstanding,
			khata_stats.customers
		FROM inv_stats, khata_stats
	`
	ops := &dto.PulseOperations{}
	if err := r.db.QueryRow(ctx, opsQuery, shopID, refDate).Scan(
		&ops.LowStockCount,
		&ops.OutOfStockCount,
		&ops.DeadStockCount,
		&ops.KhataOutstanding,
		&ops.KhataCustomers,
	); err != nil {
		r.logger.Error("failed to query dukaan pulse operations", zap.Error(err), zap.String("shop_id", shopID))
		return nil, nil, nil, err
	}

	return headline, trend, ops, nil
}

// AdvisorRawProduct holds per-product sales velocity, basket co-occurrence, and stock metrics from PostgreSQL.
type AdvisorRawProduct struct {
	ProductID    string
	Name         string
	Category     string
	SKU          string
	Price        float64
	CostPrice    float64
	CurrentStock int
	Sold30D      int
	Sold45D      int
	CoBillCount  int
	AgeDays      int
}

// GetAdvisorProductStats aggregates 30-day & 45-day product velocity, multi-item basket anchor counts,
// inventory capital, and khata receivables in PostgreSQL.
func (r *ProductRepo) GetAdvisorProductStats(ctx context.Context, shopID string, now time.Time) ([]AdvisorRawProduct, dto.CapitalSummary, error) {
	refDate := now.Format("2006-01-02")

	query := `
		WITH bill_item_counts AS (
			SELECT bill_id, COUNT(*) AS line_count
			FROM pos_bill_items
			GROUP BY bill_id
		),
		sales_agg AS (
			SELECT
				bi.product_id,
				COALESCE(SUM(CASE WHEN b.created_at >= ($2::date - INTERVAL '30 days') THEN bi.quantity ELSE 0 END), 0) AS sold_30d,
				COALESCE(SUM(bi.quantity), 0) AS sold_45d,
				COUNT(DISTINCT CASE WHEN b.created_at >= ($2::date - INTERVAL '30 days') AND bic.line_count > 1 THEN b.id END) AS co_bills
			FROM pos_bill_items bi
			JOIN pos_bills b ON b.id = bi.bill_id
			LEFT JOIN bill_item_counts bic ON bic.bill_id = b.id
			WHERE b.shop_id = $1
			  AND COALESCE(b.status, '') <> 'cancelled'
			  AND b.created_at >= ($2::date - INTERVAL '45 days')
			GROUP BY bi.product_id
		)
		SELECT
			p.id,
			p.name,
			COALESCE(p.category, ''),
			COALESCE(p.sku, ''),
			p.price,
			COALESCE(p.cost_price, 0),
			GREATEST(COALESCE(i.quantity, 0), 0),
			COALESCE(s.sold_30d, 0),
			COALESCE(s.sold_45d, 0),
			COALESCE(s.co_bills, 0),
			GREATEST(EXTRACT(DAY FROM ($2::timestamp - p.created_at))::int, 0)
		FROM products p
		LEFT JOIN inventory i ON i.product_id = p.id
		LEFT JOIN sales_agg s ON s.product_id = p.id
		WHERE p.shop_id = $1 AND p.is_active = true
	`

	rows, err := r.db.Query(ctx, query, shopID, refDate)
	if err != nil {
		r.logger.Error("failed to query advisor product stats", zap.Error(err), zap.String("shop_id", shopID))
		return nil, dto.CapitalSummary{}, err
	}
	defer rows.Close()

	var items []AdvisorRawProduct
	var capital dto.CapitalSummary
	for rows.Next() {
		var it AdvisorRawProduct
		if err := rows.Scan(
			&it.ProductID,
			&it.Name,
			&it.Category,
			&it.SKU,
			&it.Price,
			&it.CostPrice,
			&it.CurrentStock,
			&it.Sold30D,
			&it.Sold45D,
			&it.CoBillCount,
			&it.AgeDays,
		); err != nil {
			return nil, dto.CapitalSummary{}, err
		}
		capital.InventoryCapital += float64(it.CurrentStock) * it.CostPrice
		items = append(items, it)
	}

	_ = r.db.QueryRow(ctx, `
		SELECT COALESCE(SUM(CASE WHEN current_balance > 0 THEN current_balance ELSE 0 END), 0)
		FROM customer_khata
		WHERE shop_id = $1
	`, shopID).Scan(&capital.KhataReceivables)

	return items, capital, nil
}

// BulkImportProducts executes a high-speed batch insert and upsert transaction for importing lakhs of products efficiently and reliably.
