// Package dto provides request and response structures.
package dto

type CreateCategoryRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Slug        string  `json:"slug" validate:"omitempty,min=2,max=100"`
	Description string  `json:"description" validate:"omitempty,max=500"`
	ImageURL    string  `json:"image_url" validate:"omitempty,max=500"`
	ParentID    *string `json:"parent_id,omitempty" validate:"omitempty,uuid4"`
}

type UpdateCategoryRequest struct {
	Name        string  `json:"name" validate:"required,min=2,max=100"`
	Slug        string  `json:"slug" validate:"omitempty,min=2,max=100"`
	Description string  `json:"description" validate:"omitempty,max=500"`
	ImageURL    string  `json:"image_url" validate:"omitempty,max=500"`
	ParentID    *string `json:"parent_id,omitempty" validate:"omitempty,uuid4"`
	IsActive    *bool   `json:"is_active,omitempty"`
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
