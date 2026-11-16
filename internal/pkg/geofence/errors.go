package geofence

import "errors"

var (
	// ErrInvalidPolygon is returned when a polygon has fewer than 3 coordinates.
	ErrInvalidPolygon = errors.New("geofence: polygon must contain at least 3 distinct vertices")

	// ErrInvalidCoordinates is returned when latitude is outside [-90, 90] or longitude outside [-180, 180].
	ErrInvalidCoordinates = errors.New("geofence: coordinates out of valid WGS84 range")

	// ErrZoneNotFound is returned when attempting to delete or access an unmapped geofence zone.
	ErrZoneNotFound = errors.New("geofence: zone not found")
)
