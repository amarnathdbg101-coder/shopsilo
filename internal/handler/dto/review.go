// Package dto handle request and response struct.
package dto

import "shopMe/internal/handler/model"

type CreateReviewRequest struct {
	Rating  int    `json:"rating" validate:"required,min=1,max=5"`
	Comment string `json:"comment,omitempty" validate:"omitempty,max=500"`
}

type ReviewPaginationResponse struct {
	Reviews       []*model.ShopReview `json:"reviews"`
	AverageRating float64             `json:"average_rating"`
	TotalReviews  int                 `json:"total_reviews"`
	Page          int                 `json:"page"`
	Limit         int                 `json:"limit"`
	TotalPages    int                 `json:"total_pages"`
}
