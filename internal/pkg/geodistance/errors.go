package gogeodistance

import "errors"

var (
	// ErrInvalidCoordinates is returned when latitude is not in [-90, 90] or longitude is not in [-180, 180].
	ErrInvalidCoordinates = errors.New("gogeodistance: latitude must be in [-90, 90] and longitude in [-180, 180]")

	// ErrNegativeRadius is returned when radius is negative.
	ErrNegativeRadius = errors.New("gogeodistance: radius must be non-negative")

	// ErrVincentyConvergence is returned when the Vincenty formula fails to converge (antipodal points).
	ErrVincentyConvergence = errors.New("gogeodistance: vincenty formula failed to converge between antipodal points")
)
