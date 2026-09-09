// Package services handle business logic.
package services

import "errors"

var (
	// User errors
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrAccountInactive    = errors.New("account is inactive or suspended")
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidResetToken  = errors.New("invalid or expired password reset token")

	// Shop errors
	ErrUserAlreadyHasShop     = errors.New("you already have a registered shop")
	ErrShopNotFound           = errors.New("shop not found")
	ErrSlugAlreadyTaken       = errors.New("shop slug is already taken")
	ErrRestrictedRegistration = errors.New("registration restricted: device, network, or phone has been flagged for safety policy violations")

	// Moderation errors
	ErrTargetNotFound     = errors.New("reported target does not exist")
	ErrImageFlaggedBanned = errors.New("uploaded image has been flagged and banned for safety violations")

	// Product errors
	ErrProductNotFound      = errors.New("product not found")
	ErrTooManyProductImages = errors.New("a product cannot have more than 4 images")
	ErrInvalidCategory      = errors.New("specified category does not exist")

	// Upload errors
	ErrTooManyBannerImages = errors.New("cannot upload more than 2 promotional banner images for a shop")

	// Reservation errors
	ErrReservationNotFound     = errors.New("reservation not found or already processed")
	ErrMaxActiveReservations   = errors.New("you have reached the maximum allowed active reservations (limit: 3)")
	ErrProductOutOfStock       = errors.New("product does not have sufficient available stock to hold")
	ErrShopClosed              = errors.New("cannot reserve items from a shop that is currently closed")
	ErrInvalidVerificationCode = errors.New("invalid pickup code or reservation number")

	// Review errors
	ErrReviewNotFound          = errors.New("review not found")

	// Khata errors
	ErrCreditLimitExceeded     = errors.New("customer credit limit exceeded")
)
