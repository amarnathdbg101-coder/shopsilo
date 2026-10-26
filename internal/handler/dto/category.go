// Package dto provides request and response structures.
package dto

type CreateCategoryRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Slug        string  `json:"slug" validate:"omitempty,min=2,max=100"`
	Description string  `json:"description" validate:"omitempty,max=500"`
	ImageURL    string  `json:"image_url" validate:"omitempty,max=500"`
	Icon        string  `json:"icon" validate:"omitempty,max=50"`
	ParentID    *string `json:"parent_id,omitempty" validate:"omitempty,uuid4"`
}

type UpdateCategoryRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Slug        string  `json:"slug" validate:"omitempty,min=2,max=100"`
	Description string  `json:"description" validate:"omitempty,max=500"`
	ImageURL    string  `json:"image_url" validate:"omitempty,max=500"`
	Icon        string  `json:"icon" validate:"omitempty,max=50"`
	ParentID    *string `json:"parent_id,omitempty" validate:"omitempty,uuid4"`
	IsActive    *bool   `json:"is_active,omitempty"`
}

type ShopCategoryItem struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Description  string  `json:"description"`
	ImageURL     string  `json:"image_url"`
	Icon         string  `json:"icon"`
	ShopID       *string `json:"shop_id,omitempty"`
	IsCustom     bool    `json:"is_custom"`
	IsActive     bool    `json:"is_active"`
	ProductCount int     `json:"product_count"`
	CreatedAt    string  `json:"created_at"`
}

type AdminCategoryItem struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	Slug         string  `json:"slug"`
	Description  string  `json:"description"`
	ImageURL     string  `json:"image_url"`
	ParentID     *string `json:"parent_id"`
	IsActive     bool    `json:"is_active"`
	ProductCount int     `json:"product_count"`
	CreatedAt    string  `json:"created_at"`
}
