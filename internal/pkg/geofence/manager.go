package geofence

import (
	"math"
	"sync"
)

// Zone represents a named delivery or operating territory.
type Zone struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Polygon  *Polygon `json:"polygon"`
	Metadata any      `json:"metadata,omitempty"`
}

// Manager coordinates spatial indexing and point-in-polygon matching across multiple delivery zones.
type Manager struct {
	mu    sync.RWMutex
	zones map[string]*Zone
}

// NewManager creates a new geofence manager instance.
func NewManager() *Manager {
	return &Manager{
		zones: make(map[string]*Zone),
	}
}

// AddZone registers a delivery zone.
func (m *Manager) AddZone(zone *Zone) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.zones[zone.ID] = zone
}

// RemoveZone unregisters a delivery zone.
func (m *Manager) RemoveZone(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.zones[id]; !exists {
		return ErrZoneNotFound
	}
	delete(m.zones, id)
	return nil
}

// FindZone returns the first delivery zone that contains the specified GPS point.
func (m *Manager) FindZone(point Point) (*Zone, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, zone := range m.zones {
		if zone.Polygon.Contains(point) {
			return zone, true
		}
	}
	return nil, false
}

// FindAllZones returns all overlapping delivery zones containing the specified GPS point.
func (m *Manager) FindAllZones(point Point) []*Zone {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var matched []*Zone
	for _, zone := range m.zones {
		if zone.Polygon.Contains(point) {
			matched = append(matched, zone)
		}
	}
	return matched
}

// NearestZone finds the zone whose centroid is closest to the given point, returning the zone and distance in meters.
func (m *Manager) NearestZone(point Point) (*Zone, float64) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var nearest *Zone
	minDist := math.MaxFloat64

	for _, zone := range m.zones {
		d := DistanceMeters(point, zone.Polygon.Centroid)
		if d < minDist {
			minDist = d
			nearest = zone
		}
	}

	return nearest, minDist
}
