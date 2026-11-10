package reuse_test

import (
	"testing"

	"shopMe/internal/reuse"
)

func TestDeliveryGeofence_PointInPolygon(t *testing.T) {
	// Connaught Place, New Delhi delivery zone
	vertices := []reuse.DeliveryPoint{
		{Lat: 28.6300, Lon: 77.2150},
		{Lat: 28.6350, Lon: 77.2150},
		{Lat: 28.6350, Lon: 77.2250},
		{Lat: 28.6300, Lon: 77.2250},
	}

	poly, err := reuse.CreateDeliveryPolygon(vertices)
	if err != nil {
		t.Fatalf("unexpected error creating delivery polygon: %v", err)
	}

	// Point inside polygon
	inside := reuse.IsAddressInsideDeliveryZone(poly, 28.6325, 77.2200)
	if !inside {
		t.Fatalf("expected point (28.6325, 77.2200) to be inside delivery zone")
	}

	// Point far outside polygon (Noida)
	outside := reuse.IsAddressInsideDeliveryZone(poly, 28.5700, 77.3200)
	if outside {
		t.Fatalf("expected point (28.5700, 77.3200) to be outside delivery zone")
	}
}
