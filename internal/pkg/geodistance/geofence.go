package gogeodistance

import "math"

// EarthRadiusKm is the mean Earth radius in kilometers.
const EarthRadiusKm = EarthRadiusMeters / 1000.0

// BoundingBox defines a rectangular geographic area (useful for fast database indexing and filtering).
type BoundingBox struct {
	MinLat float64 `json:"min_lat"`
	MaxLat float64 `json:"max_lat"`
	MinLng float64 `json:"min_lng"`
	MaxLng float64 `json:"max_lng"`
}

// BoundingBoxFromRadius calculates the bounding coordinates enclosing a circular area of radiusKm around center.
func BoundingBoxFromRadius(center Point, radiusKm float64) BoundingBox {
	if radiusKm < 0 {
		radiusKm = 0
	}

	radDist := radiusKm / EarthRadiusKm

	minLat := center.Lat - toDegrees(radDist)
	maxLat := center.Lat + toDegrees(radDist)

	minLng := -180.0
	maxLng := 180.0

	// Check if pole is enclosed
	if minLat > -90 && maxLat < 90 {
		deltaLng := math.Asin(math.Sin(radDist) / math.Cos(toRadians(center.Lat)))
		minLng = center.Lng - toDegrees(deltaLng)
		if minLng < -180 {
			minLng += 360
		}
		maxLng = center.Lng + toDegrees(deltaLng)
		if maxLng > 180 {
			maxLng -= 360
		}
	} else {
		minLat = math.Max(minLat, -90)
		maxLat = math.Min(maxLat, 90)
		minLng = -180
		maxLng = 180
	}

	if minLat > maxLat {
		minLat, maxLat = maxLat, minLat
	}
	if minLng > maxLng {
		minLng, maxLng = maxLng, minLng
	}

	return BoundingBox{
		MinLat: minLat,
		MaxLat: maxLat,
		MinLng: minLng,
		MaxLng: maxLng,
	}
}

// Contains returns true if the point falls inside the bounding box.
func (b BoundingBox) Contains(p Point) bool {
	return p.Lat >= b.MinLat && p.Lat <= b.MaxLat &&
		p.Lng >= b.MinLng && p.Lng <= b.MaxLng
}

// IsWithinRadius checks if target point is within radiusKm of center using Haversine calculation.
func IsWithinRadius(center, target Point, radiusKm float64) bool {
	return HaversineKm(center, target) <= radiusKm
}

// PolygonContains uses the ray casting point-in-polygon algorithm to determine if p is inside a polygon.
func PolygonContains(polygon []Point, p Point) bool {
	if len(polygon) < 3 {
		return false
	}

	inside := false
	j := len(polygon) - 1

	for i := 0; i < len(polygon); i++ {
		xi, yi := polygon[i].Lng, polygon[i].Lat
		xj, yj := polygon[j].Lng, polygon[j].Lat

		intersect := ((yi > p.Lat) != (yj > p.Lat)) &&
			(p.Lng < (xj-xi)*(p.Lat-yi)/(yj-yi)+xi)

		if intersect {
			inside = !inside
		}
		j = i
	}

	return inside
}
