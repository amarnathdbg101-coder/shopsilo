package reuse_test

import (
	"math"
	"testing"

	"shopMe/internal/reuse"
)

func TestReuse_Geodistance(t *testing.T) {
	// Connaught Place, New Delhi -> India Gate, New Delhi (~2.2 km)
	cpLat, cpLng := 28.6315, 77.2167
	igLat, igLng := 28.6129, 77.2295

	km := reuse.CalculateDistanceKm(cpLat, cpLng, igLat, igLng)
	if km < 2.0 || km > 2.6 {
		t.Fatalf("expected distance ~2.4 km, got %.2f km", km)
	}

	formatted := reuse.FormatDistanceKm(km)
	if formatted != "2.4 km" {
		t.Fatalf("expected formatted distance '2.4 km', got '%s'", formatted)
	}

	// Bearing check: India Gate is South-East of Connaught Place
	deg, compass := reuse.BearingAndDirection(cpLat, cpLng, igLat, igLng)
	if deg < 120 || deg > 170 {
		t.Fatalf("expected SE bearing (~145 deg), got %.2f", deg)
	}
	if compass != "SE" && compass != "SSE" {
		t.Fatalf("expected compass direction SE or SSE, got %s", compass)
	}

	// Bounding box check
	minLat, maxLat, minLng, maxLng := reuse.BoundingBoxFromRadius(cpLat, cpLng, 5.0)
	if cpLat < minLat || cpLat > maxLat || cpLng < minLng || cpLng > maxLng {
		t.Fatal("bounding box does not contain center coordinates")
	}

	// Geofence radius check
	if !reuse.IsWithinRadius(cpLat, cpLng, igLat, igLng, 5.0) {
		t.Fatal("expected India Gate to be within 5km of Connaught Place")
	}
	if reuse.IsWithinRadius(cpLat, cpLng, 19.0760, 72.8777, 5.0) {
		t.Fatal("expected Mumbai to be outside 5km radius of Connaught Place")
	}
	_ = math.Abs
}
