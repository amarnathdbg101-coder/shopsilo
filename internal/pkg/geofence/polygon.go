package geofence

import "math"

// Polygon represents a closed geographic boundary formed by ordered vertices.
type Polygon struct {
	Vertices []Point `json:"vertices"`
	BBox     BBox    `json:"bbox"`
	Centroid Point   `json:"centroid"`
}

// NewPolygon constructs and pre-indexes a polygon with its bounding box and centroid.
func NewPolygon(vertices []Point) (*Polygon, error) {
	if len(vertices) < 3 {
		return nil, ErrInvalidPolygon
	}

	for _, v := range vertices {
		if v.Lat < -90 || v.Lat > 90 || v.Lon < -180 || v.Lon > 180 {
			return nil, ErrInvalidCoordinates
		}
	}

	bbox := BBox{
		MinLat: math.MaxFloat64,
		MaxLat: -math.MaxFloat64,
		MinLon: math.MaxFloat64,
		MaxLon: -math.MaxFloat64,
	}

	var sumLat, sumLon float64
	for _, v := range vertices {
		if v.Lat < bbox.MinLat {
			bbox.MinLat = v.Lat
		}
		if v.Lat > bbox.MaxLat {
			bbox.MaxLat = v.Lat
		}
		if v.Lon < bbox.MinLon {
			bbox.MinLon = v.Lon
		}
		if v.Lon > bbox.MaxLon {
			bbox.MaxLon = v.Lon
		}
		sumLat += v.Lat
		sumLon += v.Lon
	}

	n := float64(len(vertices))
	centroid := Point{
		Lat: sumLat / n,
		Lon: sumLon / n,
	}

	return &Polygon{
		Vertices: vertices,
		BBox:     bbox,
		Centroid: centroid,
	}, nil
}

// Contains determines if a point lies within the polygon using Ray Casting with O(1) BBox pre-filtering.
func (p *Polygon) Contains(point Point) bool {
	// Step 1: O(1) Axis-Aligned Bounding Box exclusion
	if !p.BBox.Contains(point) {
		return false
	}

	// Step 2: Ray Casting Algorithm (Jordan curve theorem)
	inside := false
	n := len(p.Vertices)
	j := n - 1

	for i := 0; i < n; i++ {
		vi := p.Vertices[i]
		vj := p.Vertices[j]

		// Check if point intersects the edge between vi and vj
		intersectCondition := ((vi.Lat > point.Lat) != (vj.Lat > point.Lat)) &&
			(point.Lon < (vj.Lon-vi.Lon)*(point.Lat-vi.Lat)/(vj.Lat-vi.Lat)+vi.Lon)

		if intersectCondition {
			inside = !inside
		}
		j = i
	}

	return inside
}
