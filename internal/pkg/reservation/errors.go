package goreservation

import "errors"

var (
	// ErrReservationNotFound is returned when operating on a non-existent reservation ID.
	ErrReservationNotFound = errors.New("goreservation: reservation not found")

	// ErrReservationExpired is returned when attempting to confirm an expired reservation.
	ErrReservationExpired = errors.New("goreservation: reservation lease has expired")

	// ErrReservationNotPending is returned when confirming or releasing a non-pending reservation.
	ErrReservationNotPending = errors.New("goreservation: reservation is not in pending status")

	// ErrInsufficientStock is returned when inventory stock is inadequate to satisfy the hold quantity.
	ErrInsufficientStock = errors.New("goreservation: insufficient inventory stock to satisfy hold")

	// ErrZeroOrNegativeQuantity is returned when an item quantity <= 0.
	ErrZeroOrNegativeQuantity = errors.New("goreservation: item quantity must be strictly positive")

	// ErrEmptyItems is returned when attempting to hold an empty items list.
	ErrEmptyItems = errors.New("goreservation: reservation items list cannot be empty")

	// ErrInvalidDuration is returned when TTL is <= 0.
	ErrInvalidDuration = errors.New("goreservation: hold duration must be greater than zero")
)
