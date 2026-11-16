# go-geodistance

[![Go Reference](https://pkg.go.dev/badge/github.com/amarnathdbg101-coder/go/go-geodistance.svg)](https://pkg.go.dev/github.com/amarnathdbg101-coder/go/go-geodistance)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

High-performance, zero-dependency geospatial calculation toolkit for Go. Fast Haversine & Vincenty geodesic calculations, bounding box generation, polygonal geofencing, and compass bearing navigation.

---

## Features
- **Haversine Formula**: Ultra-fast spherical great-circle distance in meters and kilometers.
- **Vincenty WGS-84**: High-precision ellipsoidal geodesic distance accurate to 0.5 mm worldwide.
- **Bounding Box Generator**: Fast database pre-filtering (`MinLat`, `MaxLat`, `MinLng`, `MaxLng`) for radius queries.
- **Ray-Casting Polygon Geofencing**: Check if coordinates are inside delivery zones, service areas, or custom polygons.
- **Forward Azimuth & Compass Directions**: 16-point cardinal compass direction mapping ("N", "NE", "SSW").
- **E-Commerce Distance Formatter**: Standardized readable distance labels ("450 m", "1.2 km").
- **Zero Third-Party Dependencies**: 100% pure Go standard library (`math`).

---

## Installation

```bash
go get github.com/amarnathdbg101-coder/go/go-geodistance@v1.0.0
```

---

## Quickstart

### 1. Distance Calculation

```go
delhi := gogeodistance.MustPoint(28.6139, 77.2090)
mumbai := gogeodistance.MustPoint(19.0760, 72.8777)

// Haversine (Fast)
km := gogeodistance.HaversineKm(delhi, mumbai)
fmt.Printf("Distance: %.2f km\n", km) // ~1148.5 km

// Vincenty (Sub-millimeter Ellipsoidal Precision)
vincentyKm, _ := gogeodistance.VincentyKm(delhi, mumbai)
fmt.Printf("Ellipsoidal Distance: %.2f km\n", vincentyKm)
```

### 2. Bounding Box for SQL Queries

```go
userLocation := gogeodistance.MustPoint(lat, lng)
bbox := gogeodistance.BoundingBoxFromRadius(userLocation, 5.0) // 5 km box

// Use in B-Tree indexed SQL query without needing PostGIS
query := `
    SELECT id, name, latitude, longitude
    FROM shops
    WHERE latitude BETWEEN $1 AND $2
      AND longitude BETWEEN $3 AND $4
`
rows, err := db.Query(ctx, query, bbox.MinLat, bbox.MaxLat, bbox.MinLng, bbox.MaxLng)
```

### 3. Compass Bearing & Delivery Formatting

```go
bearing := gogeodistance.Bearing(storePoint, customerPoint)
direction := gogeodistance.CompassDirection(bearing)
label := gogeodistance.FormatDistance(gogeodistance.Haversine(storePoint, customerPoint))

fmt.Printf("%s away towards %s\n", label, direction) // "1.2 km away towards NE"
```

---

## License
MIT License.
