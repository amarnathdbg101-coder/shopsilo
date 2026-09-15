package model

import "time"

type ProductMediaVaultItem struct {
	ID               string    `json:"id"`
	ProductCode      string    `json:"product_code"`
	ContentHash      string    `json:"content_hash"`
	ImageURL         string    `json:"image_url"`
	OriginalFilename string    `json:"original_filename,omitempty"`
	MimeType         string    `json:"mime_type,omitempty"`
	FileSizeBytes    int64     `json:"file_size_bytes"`
	UploaderShopID   *string   `json:"uploader_shop_id,omitempty"`
	IsVerifiedMaster bool      `json:"is_verified_master"`
	ReferenceCount   int       `json:"reference_count"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
