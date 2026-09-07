// Package model handle database struct.
package model

import "time"

type ShopReview struct {
	ID                string    `json:"id"`
	ShopID            string    `json:"shop_id"`
	UserID            string    `json:"user_id"`
	UserName          string    `json:"user_name,omitempty"`
	Rating            int       `json:"rating"` // 1 to 5
	Comment           string    `json:"comment,omitempty"`
	IsVerifiedVisitor bool      `json:"is_verified_visitor"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ShopRatingStats struct {
	AverageRating float64 `json:"average_rating"`
	TotalReviews  int     `json:"total_reviews"`
}
