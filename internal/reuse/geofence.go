// Package reuse provides reusable foundational wrappers and shared infrastructure utilities.
package reuse

import (
	"errors"

	"github.com/amarnathdbg101-coder/go/go-geofence"
)

// DeliveryPoint represents a GPS location for geofencing.
type DeliveryPoint = geofence.Point

// DeliveryZone represents a merchant's custom polygonal delivery perimeter.
type DeliveryZone = geofence.Zone

// GlobalZoneManager coordinates delivery zones across active merchant clusters.
var GlobalZoneManager = geofence.NewManager()

// CreateDeliveryPolygon constructs a validated delivery boundary from GPS vertices.
func CreateDeliveryPolygon(vertices []DeliveryPoint) (*geofence.Polygon, error) {
	if len(vertices) < 3 {
		return nil, errors.New("delivery zone polygon requires at least 3 GPS boundary points")
	}
	return geofence.NewPolygon(vertices)
}

// IsAddressInsideDeliveryZone verifies if customer coordinates fall inside the specified delivery polygon.
func IsAddressInsideDeliveryZone(polygon *geofence.Polygon, customerLat, customerLon float64) bool {
	if polygon == nil {
		return false
	}
	return polygon.Contains(geofence.Point{Lat: customerLat, Lon: customerLon})
}

// RegisterMerchantDeliveryZone adds a custom delivery zone to the global spatial manager.
func RegisterMerchantDeliveryZone(shopID, shopName string, vertices []DeliveryPoint) (*DeliveryZone, error) {
	poly, err := CreateDeliveryPolygon(vertices)
	if err != nil {
		return nil, err
	}

	zone := &geofence.Zone{
		ID:      shopID,
		Name:    shopName,
		Polygon: poly,
	}

	GlobalZoneManager.AddZone(zone)
	return zone, nil
}
