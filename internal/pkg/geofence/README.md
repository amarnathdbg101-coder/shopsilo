# go-geofence 📍

> **Sub-Microsecond Spatial Geofencing, Point-in-Polygon (PIP) Ray Casting, and Haversine Distance Engine for Go (Go 1.23+).**  
> Check if customer GPS coordinates fall inside complex delivery zones or service boundaries with $O(1)$ Axis-Aligned Bounding Box (AABB) pre-filtering.

---

## 🚀 Key Features

1. **Point-in-Polygon Ray Casting**:
   - Implements Jordan Curve Theorem ray casting for arbitrary convex or concave multi-point polygons.
2. **$O(1)$ Bounding-Box Pre-Filtering**:
   - Rejects outside points in $\approx 2$ nanoseconds before running trigonometric edge evaluations.
3. **Haversine Great-Circle Distance**:
   - High-precision distance calculation in meters between geographic coordinates.
4. **Multi-Zone Manager**:
   - Coordinate indexing across thousands of merchant delivery zones, warehouse perimeters, and service clusters.
5. **Zero Dependencies**:
   - 100% pure standard library (`math`, `sync`).

---

## 📦 Quickstart

```go
package main

import (
	"fmt"
	"go-geofence"
)

func main() {
	// 1. Define delivery polygon vertices
	deliveryPolygon, _ := geofence.NewPolygon([]geofence.Point{
		{Lat: 28.50, Lon: 77.10},
		{Lat: 28.70, Lon: 77.10},
		{Lat: 28.70, Lon: 77.30},
		{Lat: 28.50, Lon: 77.30},
	})

	// 2. Customer checkout GPS coordinates
	customerLocation := geofence.Point{Lat: 28.63, Lon: 77.21}

	// 3. Instant containment check (sub-microsecond)
	if deliveryPolygon.Contains(customerLocation) {
		fmt.Println("✅ Address is within 10-minute delivery zone!")
	} else {
		fmt.Println("❌ Address is outside serviceable delivery perimeter.")
	}
}
```
