package gogeodistance

import (
	"fmt"
	"math"
)

// EarthRadiusMeters is the volumetric mean radius of the Earth in meters (IUGG).
const EarthRadiusMeters = 6371008.8

// Point represents a geographical latitude and longitude in decimal degrees (WGS-84).
type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

// NewPoint creates and validates a Point.
func NewPoint(lat, lng float64) (Point, error) {
	if lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return Point{}, ErrInvalidCoordinates
	}
	return Point{Lat: lat, Lng: lng}, nil
}

// MustPoint creates a Point or panics if invalid. Ideal for constants and tests.
func MustPoint(lat, lng float64) Point {
	p, err := NewPoint(lat, lng)
	if err != nil {
		panic(err)
	}
	return p
}

// String returns formatted coordinates.
func (p Point) String() string {
	return fmt.Sprintf("(%.6f, %.6f)", p.Lat, p.Lng)
}

// toRadians converts degrees to radians.
func toRadians(deg float64) float64 {
	return deg * math.Pi / 180.0
}

// toDegrees converts radians to degrees.
func toDegrees(rad float64) float64 {
	return rad * 180.0 / math.Pi
}
