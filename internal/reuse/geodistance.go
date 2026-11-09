package reuse

import (
	gogeodistance "github.com/amarnathdbg101-coder/go/go-geodistance"
)

// GeoPoint represents geographic latitude and longitude.
type GeoPoint = gogeodistance.Point

// CalculateDistanceKm calculates spherical distance between two coordinates in kilometers.
func CalculateDistanceKm(lat1, lng1, lat2, lng2 float64) float64 {
	p1 := gogeodistance.Point{Lat: lat1, Lng: lng1}
	p2 := gogeodistance.Point{Lat: lat2, Lng: lng2}
	return gogeodistance.HaversineKm(p1, p2)
}

// CalculateEllipsoidalDistanceKm calculates high-precision WGS-84 geodesic distance in kilometers.
func CalculateEllipsoidalDistanceKm(lat1, lng1, lat2, lng2 float64) (float64, error) {
	p1 := gogeodistance.Point{Lat: lat1, Lng: lng1}
	p2 := gogeodistance.Point{Lat: lat2, Lng: lng2}
	return gogeodistance.VincentyKm(p1, p2)
}

// FormatDistance formats meters into human-readable delivery distance (e.g. "450 m", "1.2 km").
func FormatDistance(meters float64) string {
	return gogeodistance.FormatDistance(meters)
}

// FormatDistanceKm formats kilometers into human-readable delivery distance.
func FormatDistanceKm(km float64) string {
	return gogeodistance.FormatDistance(km * 1000.0)
}

// BoundingBoxFromRadius generates (minLat, maxLat, minLng, maxLng) for indexed SQL spatial pre-filtering.
func BoundingBoxFromRadius(lat, lng, radiusKm float64) (minLat, maxLat, minLng, maxLng float64) {
	center := gogeodistance.Point{Lat: lat, Lng: lng}
	bbox := gogeodistance.BoundingBoxFromRadius(center, radiusKm)
	return bbox.MinLat, bbox.MaxLat, bbox.MinLng, bbox.MaxLng
}

// IsWithinRadius checks if a coordinate is within the circular radius of another coordinate.
func IsWithinRadius(centerLat, centerLng, targetLat, targetLng, radiusKm float64) bool {
	center := gogeodistance.Point{Lat: centerLat, Lng: centerLng}
	target := gogeodistance.Point{Lat: targetLat, Lng: targetLng}
	return gogeodistance.IsWithinRadius(center, target, radiusKm)
}

// BearingAndDirection computes the forward azimuth and 16-point cardinal compass direction (e.g. "NE", "SSW").
func BearingAndDirection(fromLat, fromLng, toLat, toLng float64) (degrees float64, compass string) {
	from := gogeodistance.Point{Lat: fromLat, Lng: fromLng}
	to := gogeodistance.Point{Lat: toLat, Lng: toLng}
	deg := gogeodistance.Bearing(from, to)
	return deg, gogeodistance.CompassDirection(deg)
}
