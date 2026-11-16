# Technical Specification: go-geodistance

## 1. Domain & Purpose
`go-geodistance` is a high-performance, zero-dependency geospatial calculation toolkit for Go. It provides accurate great-circle calculations (Haversine formula), sub-millimeter ellipsoidal geodesics (Vincenty WGS-84 formula), spatial bounding-box generators for database queries, polygonal geofence validation (Ray Casting), and compass bearing calculations.

## 2. Architecture & Design Principles
- **Dual Geodesic Algorithms**:
  - `Haversine`: Extremely fast spherical trigonometry, ideal for real-time mobile sorting, delivery filters, and in-memory searches.
  - `Vincenty`: High-precision iterative solution on the WGS-84 reference ellipsoid ($a=6378137.0\text{ m}, 1/f=298.257223563$) accurate to 0.5 mm worldwide.
- **Bounding Box Pre-Filtering**: Generates `(MinLat, MaxLat, MinLng, MaxLng)` limits for circular search radii to enable $O(\log N)$ spatial indexing on standard B-Tree columns without requiring heavy GIS extensions.
- **Geofencing & Polygon Containment**: Implements the Ray Casting algorithm for checking arbitrary store delivery zones, city limits, or geofences.
- **Navigation & Bearings**: Computes forward azimuth [0°, 360°) and maps to standard 16-point cardinal compass directions (e.g. "N", "NE", "SSW").
- **Zero Third-Party Dependencies**: 100% pure Go standard library (`math`, `fmt`).

## 3. API Surface
- `type Point struct { Lat, Lng float64 }`
- `func NewPoint(lat, lng float64) (Point, error)`
- `func MustPoint(lat, lng float64) Point`
- `func Haversine(p1, p2 Point) float64` (meters)
- `func HaversineKm(p1, p2 Point) float64` (km)
- `func Vincenty(p1, p2 Point) (float64, error)` (meters)
- `func VincentyKm(p1, p2 Point) (float64, error)` (km)
- `func Bearing(p1, p2 Point) float64` (degrees)
- `func CompassDirection(bearingDeg float64) string`
- `func FormatDistance(meters float64) string`
- `type BoundingBox struct { MinLat, MaxLat, MinLng, MaxLng float64 }`
- `func BoundingBoxFromRadius(center Point, radiusKm float64) BoundingBox`
- `func (b BoundingBox) Contains(p Point) bool`
- `func IsWithinRadius(center, target Point, radiusKm float64) bool`
- `func PolygonContains(polygon []Point, p Point) bool`
