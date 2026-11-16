package geofence

import "math"

// Point represents a WGS84 geographic coordinate (Latitude, Longitude).
type Point struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// BBox represents an Axis-Aligned Bounding Box (AABB) for instant O(1) exclusion.
type BBox struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLon float64 `json:"min_lon"`
	MaxLon float64 `json:"max_lon"`
}

// Contains checks if point is within the bounding box boundaries.
func (b BBox) Contains(p Point) bool {
	return p.Lat >= b.MinLat && p.Lat <= b.MaxLat && p.Lon >= b.MinLon && p.Lon <= b.MaxLon
}

const earthRadiusMeters = 6371000.0

// DistanceMeters computes the Haversine great-circle distance between two GPS points in meters.
func DistanceMeters(p1, p2 Point) float64 {
	dLat := (p2.Lat - p1.Lat) * (math.Pi / 180.0)
	dLon := (p2.Lon - p1.Lon) * (math.Pi / 180.0)

	lat1 := p1.Lat * (math.Pi / 180.0)
	lat2 := p2.Lat * (math.Pi / 180.0)

	a := math.Sin(dLat/2.0)*math.Sin(dLat/2.0) +
		math.Cos(lat1)*math.Cos(lat2)*math.Sin(dLon/2.0)*math.Sin(dLon/2.0)
	c := 2.0 * math.Atan2(math.Sqrt(a), math.Sqrt(1.0-a))

	return earthRadiusMeters * c
}
