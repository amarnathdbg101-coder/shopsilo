package repository

import (
	"context"
	"errors"
	"shopMe/internal/handler/model"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MediaVaultRepo struct {
	db *pgxpool.Pool
}

func NewMediaVaultRepo(db *pgxpool.Pool) *MediaVaultRepo {
	return &MediaVaultRepo{db: db}
}

// FindByContentHash checks if an identical image file has already been uploaded across the platform
func (r *MediaVaultRepo) FindByContentHash(ctx context.Context, hash string) (*model.ProductMediaVaultItem, error) {
	query := `
		SELECT id, product_code, content_hash, image_url, original_filename, mime_type, 
		       file_size_bytes, uploader_shop_id, is_verified_master, reference_count, created_at, updated_at
		FROM product_media_vault
		WHERE content_hash = $1
		ORDER BY is_verified_master DESC, created_at ASC
		LIMIT 1
	`
	var item model.ProductMediaVaultItem
	err := r.db.QueryRow(ctx, query, hash).Scan(
		&item.ID,
		&item.ProductCode,
		&item.ContentHash,
		&item.ImageURL,
		&item.OriginalFilename,
		&item.MimeType,
		&item.FileSizeBytes,
		&item.UploaderShopID,
		&item.IsVerifiedMaster,
		&item.ReferenceCount,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

// FindByProductCode finds master media assets associated with a specific Barcode / SKU
func (r *MediaVaultRepo) FindByProductCode(ctx context.Context, code string) ([]model.ProductMediaVaultItem, error) {
	cleanCode := strings.TrimSpace(strings.ToUpper(code))
	if cleanCode == "" {
		return []model.ProductMediaVaultItem{}, nil
	}

	query := `
		SELECT id, product_code, content_hash, image_url, original_filename, mime_type, 
		       file_size_bytes, uploader_shop_id, is_verified_master, reference_count, created_at, updated_at
		FROM product_media_vault
		WHERE UPPER(product_code) = $1
		ORDER BY is_verified_master DESC, reference_count DESC, created_at ASC
		LIMIT 10
	`
	rows, err := r.db.Query(ctx, query, cleanCode)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ProductMediaVaultItem
	for rows.Next() {
		var item model.ProductMediaVaultItem
		if err := rows.Scan(
			&item.ID,
			&item.ProductCode,
			&item.ContentHash,
			&item.ImageURL,
			&item.OriginalFilename,
			&item.MimeType,
			&item.FileSizeBytes,
			&item.UploaderShopID,
			&item.IsVerifiedMaster,
			&item.ReferenceCount,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// SuggestImages searches the media vault by barcode/SKU code or matching product name keywords
func (r *MediaVaultRepo) SuggestImages(ctx context.Context, code string, name string) ([]model.ProductMediaVaultItem, error) {
	cleanCode := strings.TrimSpace(strings.ToUpper(code))
	cleanName := strings.TrimSpace(strings.ToLower(name))

	query := `
		SELECT id, product_code, content_hash, image_url, original_filename, mime_type, 
		       file_size_bytes, uploader_shop_id, is_verified_master, reference_count, created_at, updated_at
		FROM product_media_vault
		WHERE ($1 != '' AND UPPER(product_code) = $1)
		   OR ($2 != '' AND (LOWER(original_filename) LIKE '%' || $2 || '%' OR LOWER(product_code) LIKE '%' || $2 || '%'))
		ORDER BY is_verified_master DESC, reference_count DESC, created_at ASC
		LIMIT 8
	`
	rows, err := r.db.Query(ctx, query, cleanCode, cleanName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []model.ProductMediaVaultItem
	for rows.Next() {
		var item model.ProductMediaVaultItem
		if err := rows.Scan(
			&item.ID,
			&item.ProductCode,
			&item.ContentHash,
			&item.ImageURL,
			&item.OriginalFilename,
			&item.MimeType,
			&item.FileSizeBytes,
			&item.UploaderShopID,
			&item.IsVerifiedMaster,
			&item.ReferenceCount,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

// UpsertVaultAsset records a new asset in the vault or increments reference count if content hash matches
func (r *MediaVaultRepo) UpsertVaultAsset(ctx context.Context, item *model.ProductMediaVaultItem) error {
	query := `
		INSERT INTO product_media_vault (
			product_code, content_hash, image_url, original_filename, 
			mime_type, file_size_bytes, uploader_shop_id, is_verified_master, reference_count, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 1, NOW(), NOW())
		ON CONFLICT (content_hash) DO UPDATE SET
			reference_count = product_media_vault.reference_count + 1,
			updated_at = NOW()
		RETURNING id, reference_count
	`
	return r.db.QueryRow(
		ctx,
		query,
		strings.TrimSpace(strings.ToUpper(item.ProductCode)),
		item.ContentHash,
		item.ImageURL,
		item.OriginalFilename,
		item.MimeType,
		item.FileSizeBytes,
		item.UploaderShopID,
		item.IsVerifiedMaster,
	).Scan(&item.ID, &item.ReferenceCount)
}

// IncrementReferenceCount increases the reference counter when a shop links to an existing image
func (r *MediaVaultRepo) IncrementReferenceCount(ctx context.Context, imageURL string) error {
	query := `
		UPDATE product_media_vault
		SET reference_count = reference_count + 1, updated_at = NOW()
		WHERE image_url = $1
	`
	_, err := r.db.Exec(ctx, query, imageURL)
	return err
}

// GetActiveUsageCount scans all shops in the products table to check how many active products still use this image
func (r *MediaVaultRepo) GetActiveUsageCount(ctx context.Context, imageURL string) (int, error) {
	query := `
		SELECT COUNT(*) 
		FROM products 
		WHERE images ? $1 AND is_active = true
	`
	var count int
	err := r.db.QueryRow(ctx, query, imageURL).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// BatchFindImagesByCodes looks up master image URLs for a list of barcodes/SKUs (used in Bulk CSV Import)
func (r *MediaVaultRepo) BatchFindImagesByCodes(ctx context.Context, codes []string) (map[string][]string, error) {
	if len(codes) == 0 {
		return map[string][]string{}, nil
	}

	cleanCodes := make([]string, 0, len(codes))
	for _, c := range codes {
		trimmed := strings.TrimSpace(strings.ToUpper(c))
		if trimmed != "" {
			cleanCodes = append(cleanCodes, trimmed)
		}
	}

	if len(cleanCodes) == 0 {
		return map[string][]string{}, nil
	}

	query := `
		SELECT UPPER(product_code), image_url 
		FROM product_media_vault 
		WHERE UPPER(product_code) = ANY($1)
		ORDER BY is_verified_master DESC, reference_count DESC
	`
	rows, err := r.db.Query(ctx, query, cleanCodes)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]string)
	for rows.Next() {
		var code, imageURL string
		if err := rows.Scan(&code, &imageURL); err != nil {
			return nil, err
		}
		result[code] = append(result[code], imageURL)
	}
	return result, nil
}
