# SPEC: go-geofence

## 1. Scope & Purpose
`go-geofence` performs spatial geometry computations for geographic boundaries, delivery zone verification, and distance calculations on the WGS84 coordinate reference system.

It does NOT provide map rendering or GeoJSON visual layers.

## 2. Public API Contract
```go
type Point struct { Lat, Lon float64 }
type BBox struct { MinLat, MaxLat, MinLon, MaxLon float64 }
type Polygon struct { ... }
type Zone struct { ... }
type Manager struct { ... }

func NewPolygon(vertices []Point) (*Polygon, error)
func (p *Polygon) Contains(point Point) bool
func DistanceMeters(p1, p2 Point) float64
func NewManager() *Manager
func (m *Manager) AddZone(zone *Zone)
func (m *Manager) FindZone(point Point) (*Zone, bool)
```

## 3. Concurrency & Memory Guarantees
- Thread-safe via `sync.RWMutex`.
- Pre-computes BBox and centroid on polygon initialization to minimize per-request allocations.
- Ray casting executes in constant stack memory without heap escapes.

## 4. How to Extend
- Add QuadTree or R-Tree spatial indexing for $> 50,000$ dense overlapping zones.
- Add GeoJSON parser importer.
