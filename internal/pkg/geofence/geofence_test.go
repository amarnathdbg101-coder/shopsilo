package geofence_test

import (
	"math"
	"testing"

	"shopMe/internal/pkg/geofence"
)

func TestPointInPolygonRayCasting(t *testing.T) {
	// A simple square polygon around New Delhi area:
	// (28.5, 77.1) to (28.7, 77.3)
	vertices := []geofence.Point{
		{Lat: 28.50, Lon: 77.10},
		{Lat: 28.70, Lon: 77.10},
		{Lat: 28.70, Lon: 77.30},
		{Lat: 28.50, Lon: 77.30},
	}

	poly, err := geofence.NewPolygon(vertices)
	if err != nil {
		t.Fatalf("failed to create polygon: %v", err)
	}

	// Point inside: Connaught Place ~ (28.63, 77.21)
	insidePoint := geofence.Point{Lat: 28.63, Lon: 77.21}
	if !poly.Contains(insidePoint) {
		t.Errorf("expected insidePoint to be inside polygon")
	}

	// Point outside: Gurgaon ~ (28.45, 77.02)
	outsidePoint := geofence.Point{Lat: 28.45, Lon: 77.02}
	if poly.Contains(outsidePoint) {
		t.Errorf("expected outsidePoint to be outside polygon")
	}
}

func TestHaversineDistance(t *testing.T) {
	// Connaught Place to India Gate (~2.3 km)
	cp := geofence.Point{Lat: 28.6315, Lon: 77.2167}
	indiaGate := geofence.Point{Lat: 28.6129, Lon: 77.2295}

	dist := geofence.DistanceMeters(cp, indiaGate)

	// Distance should be approximately 2400 meters (+- 200m)
	if math.Abs(dist-2400) > 300 {
		t.Fatalf("expected approx 2400m distance, got %.2fm", dist)
	}
}

func TestManagerFindZone(t *testing.T) {
	mgr := geofence.NewManager()

	poly, _ := geofence.NewPolygon([]geofence.Point{
		{Lat: 10.0, Lon: 10.0},
		{Lat: 20.0, Lon: 10.0},
		{Lat: 20.0, Lon: 20.0},
		{Lat: 10.0, Lon: 20.0},
	})

	zone := &geofence.Zone{
		ID:      "zone_warehouse_1",
		Name:    "Central Warehouse",
		Polygon: poly,
	}
	mgr.AddZone(zone)

	found, ok := mgr.FindZone(geofence.Point{Lat: 15.0, Lon: 15.0})
	if !ok || found.ID != "zone_warehouse_1" {
		t.Fatalf("expected to find zone_warehouse_1, got %v (ok=%v)", found, ok)
	}

	_, ok = mgr.FindZone(geofence.Point{Lat: 5.0, Lon: 5.0})
	if ok {
		t.Fatalf("expected no zone for distant point")
	}
}

