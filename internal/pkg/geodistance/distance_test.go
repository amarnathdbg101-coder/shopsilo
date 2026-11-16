package gogeodistance_test

import (
	"math"
	"testing"

	gogeodistance "shopMe/internal/pkg/geodistance"
)

func TestPoint_Validation(t *testing.T) {
	_, err := gogeodistance.NewPoint(91.0, 77.0)
	if err != gogeodistance.ErrInvalidCoordinates {
		t.Fatalf("expected ErrInvalidCoordinates for lat > 90, got %v", err)
	}

	_, err = gogeodistance.NewPoint(28.0, 185.0)
	if err != gogeodistance.ErrInvalidCoordinates {
		t.Fatalf("expected ErrInvalidCoordinates for lng > 180, got %v", err)
	}

	p, err := gogeodistance.NewPoint(28.6139, 77.2090)
	if err != nil {
		t.Fatalf("unexpected error for valid point: %v", err)
	}
	if p.String() != "(28.613900, 77.209000)" {
		t.Fatalf("unexpected string output: %s", p.String())
	}
}

func TestHaversineAndVincenty_DelhiToMumbai(t *testing.T) {
	delhi := gogeodistance.MustPoint(28.6139, 77.2090)
	mumbai := gogeodistance.MustPoint(19.0760, 72.8777)

	haversineKm := gogeodistance.HaversineKm(delhi, mumbai)
	vincentyKm, err := gogeodistance.VincentyKm(delhi, mumbai)
	if err != nil {
		t.Fatalf("vincenty failed: %v", err)
	}

	// Great circle distance between Delhi and Mumbai is approx 1,148 km
	if haversineKm < 1140 || haversineKm > 1160 {
		t.Fatalf("expected haversine distance ~1,148 km, got %.2f km", haversineKm)
	}

	if vincentyKm < 1140 || vincentyKm > 1160 {
		t.Fatalf("expected vincenty distance ~1,148 km, got %.2f km", vincentyKm)
	}

	// Difference between spherical Haversine and ellipsoidal Vincenty should be within 0.5%
	diff := math.Abs(haversineKm - vincentyKm)
	if diff > 10.0 {
		t.Fatalf("difference between Haversine and Vincenty too large: %.2f km", diff)
	}
}

func TestBearingAndCompass(t *testing.T) {
	p1 := gogeodistance.MustPoint(0, 0)
	north := gogeodistance.MustPoint(10, 0)
	east := gogeodistance.MustPoint(0, 10)
	south := gogeodistance.MustPoint(-10, 0)
	west := gogeodistance.MustPoint(0, -10)

	bN := gogeodistance.Bearing(p1, north)
	if math.Abs(bN-0.0) > 0.01 && math.Abs(bN-360.0) > 0.01 {
		t.Fatalf("expected bearing ~0 deg North, got %.2f", bN)
	}
	if gogeodistance.CompassDirection(bN) != "N" {
		t.Fatalf("expected compass 'N', got %s", gogeodistance.CompassDirection(bN))
	}

	bE := gogeodistance.Bearing(p1, east)
	if math.Abs(bE-90.0) > 0.01 {
		t.Fatalf("expected bearing ~90 deg East, got %.2f", bE)
	}
	if gogeodistance.CompassDirection(bE) != "E" {
		t.Fatalf("expected compass 'E', got %s", gogeodistance.CompassDirection(bE))
	}

	bS := gogeodistance.Bearing(p1, south)
	if math.Abs(bS-180.0) > 0.01 {
		t.Fatalf("expected bearing ~180 deg South, got %.2f", bS)
	}
	if gogeodistance.CompassDirection(bS) != "S" {
		t.Fatalf("expected compass 'S', got %s", gogeodistance.CompassDirection(bS))
	}

	bW := gogeodistance.Bearing(p1, west)
	if math.Abs(bW-270.0) > 0.01 {
		t.Fatalf("expected bearing ~270 deg West, got %.2f", bW)
	}
	if gogeodistance.CompassDirection(bW) != "W" {
		t.Fatalf("expected compass 'W', got %s", gogeodistance.CompassDirection(bW))
	}
}

func TestFormatDistance(t *testing.T) {
	cases := []struct {
		meters   float64
		expected string
	}{
		{450, "450 m"},
		{999, "999 m"},
		{1200, "1.2 km"},
		{5780, "5.8 km"},
		{18500, "18 km"},
	}

	for _, tc := range cases {
		actual := gogeodistance.FormatDistance(tc.meters)
		if actual != tc.expected {
			t.Fatalf("for %.0f meters expected %s, got %s", tc.meters, tc.expected, actual)
		}
	}
}

func TestBoundingBoxAndRadius(t *testing.T) {
	center := gogeodistance.MustPoint(28.6139, 77.2090)
	bbox := gogeodistance.BoundingBoxFromRadius(center, 5.0) // 5 km box

	// Center must be contained
	if !bbox.Contains(center) {
		t.Fatal("bounding box must contain center point")
	}

	// Point 2km away should be within radius and within bounding box
	nearby := gogeodistance.MustPoint(28.6250, 77.2150)
	if !gogeodistance.IsWithinRadius(center, nearby, 5.0) {
		t.Fatal("expected nearby point to be within 5km radius")
	}
	if !bbox.Contains(nearby) {
		t.Fatal("expected nearby point to be within bounding box")
	}

	// Point 50km away should NOT be within radius
	far := gogeodistance.MustPoint(28.2000, 77.2090)
	if gogeodistance.IsWithinRadius(center, far, 5.0) {
		t.Fatal("expected far point to be outside 5km radius")
	}
}

func TestPolygonContains(t *testing.T) {
	// Simple square polygon around (0,0) with size 2x2
	square := []gogeodistance.Point{
		gogeodistance.MustPoint(-1, -1),
		gogeodistance.MustPoint(1, -1),
		gogeodistance.MustPoint(1, 1),
		gogeodistance.MustPoint(-1, 1),
	}

	inside := gogeodistance.MustPoint(0, 0)
	outside := gogeodistance.MustPoint(2, 2)

	if !gogeodistance.PolygonContains(square, inside) {
		t.Fatal("expected (0,0) to be inside square polygon")
	}
	if gogeodistance.PolygonContains(square, outside) {
		t.Fatal("expected (2,2) to be outside square polygon")
	}
}
