package gogeodistance

import (
	"fmt"
	"math"
)

// WGS-84 ellipsoid parameters
const (
	wgs84A = 6378137.0           // Semi-major axis (meters)
	wgs84F = 1.0 / 298.257223563 // Flattening
	wgs84B = 6356752.314245      // Semi-minor axis (meters) = (1 - f) * a
)

// Haversine calculates great-circle distance between two points in meters using the spherical law of haversines.
func Haversine(p1, p2 Point) float64 {
	lat1Rad := toRadians(p1.Lat)
	lat2Rad := toRadians(p2.Lat)
	dLat := toRadians(p2.Lat - p1.Lat)
	dLng := toRadians(p2.Lng - p1.Lng)

	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(h), math.Sqrt(1-h))
	return EarthRadiusMeters * c
}

// HaversineKm calculates great-circle distance in kilometers.
func HaversineKm(p1, p2 Point) float64 {
	return Haversine(p1, p2) / 1000.0
}

// Vincenty calculates high-precision geodesic distance between two points on the WGS-84 ellipsoid in meters.
// Accurate to 0.5 mm worldwide.
func Vincenty(p1, p2 Point) (float64, error) {
	if p1.Lat == p2.Lat && p1.Lng == p2.Lng {
		return 0.0, nil
	}

	phi1 := toRadians(p1.Lat)
	phi2 := toRadians(p2.Lat)
	L := toRadians(p2.Lng - p1.Lng)

	U1 := math.Atan((1 - wgs84F) * math.Tan(phi1))
	U2 := math.Atan((1 - wgs84F) * math.Tan(phi2))

	sinU1, cosU1 := math.Sin(U1), math.Cos(U1)
	sinU2, cosU2 := math.Sin(U2), math.Cos(U2)

	lambda := L
	var lambdaP float64
	var sinSigma, cosSigma, sigma, sinAlpha, cos2Alpha, cos2SigmaM float64

	const maxIter = 200
	iter := 0

	for {
		sinLambda, cosLambda := math.Sin(lambda), math.Cos(lambda)
		sinSigma = math.Sqrt((cosU2*sinLambda)*(cosU2*sinLambda) +
			(cosU1*sinU2-sinU1*cosU2*cosLambda)*(cosU1*sinU2-sinU1*cosU2*cosLambda))

		if sinSigma == 0 {
			return 0.0, nil // Coincident points
		}

		cosSigma = sinU1*sinU2 + cosU1*cosU2*cosLambda
		sigma = math.Atan2(sinSigma, cosSigma)

		sinAlpha = (cosU1 * cosU2 * sinLambda) / sinSigma
		cos2Alpha = 1 - sinAlpha*sinAlpha

		if cos2Alpha == 0 {
			cos2SigmaM = 0 // Equatorial line
		} else {
			cos2SigmaM = cosSigma - 2*sinU1*sinU2/cos2Alpha
		}

		C := (wgs84F / 16) * cos2Alpha * (4 + wgs84F*(4-3*cos2Alpha))
		lambdaP = lambda
		lambda = L + (1-C)*wgs84F*sinAlpha*(sigma+C*sinSigma*(cos2SigmaM+C*cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)))

		iter++
		if math.Abs(lambda-lambdaP) < 1e-12 || iter > maxIter {
			break
		}
	}

	if iter > maxIter {
		return 0, ErrVincentyConvergence
	}

	uSq := cos2Alpha * (wgs84A*wgs84A - wgs84B*wgs84B) / (wgs84B * wgs84B)
	A := 1 + (uSq/16384)*(4096+uSq*(-768+uSq*(320-175*uSq)))
	B := (uSq / 1024) * (256 + uSq*(-128+uSq*(74-47*uSq)))
	deltaSigma := B * sinSigma * (cos2SigmaM + (B/4)*(cosSigma*(-1+2*cos2SigmaM*cos2SigmaM)-
		(B/6)*cos2SigmaM*(-3+4*sinSigma*sinSigma)*(-3+4*cos2SigmaM*cos2SigmaM)))

	s := wgs84B * A * (sigma - deltaSigma)
	return s, nil
}

// VincentyKm calculates Vincenty distance in kilometers.
func VincentyKm(p1, p2 Point) (float64, error) {
	meters, err := Vincenty(p1, p2)
	if err != nil {
		return 0, err
	}
	return meters / 1000.0, nil
}

// Bearing calculates the initial compass bearing (forward azimuth) from p1 to p2 in degrees [0, 360).
func Bearing(p1, p2 Point) float64 {
	lat1Rad := toRadians(p1.Lat)
	lat2Rad := toRadians(p2.Lat)
	dLng := toRadians(p2.Lng - p1.Lng)

	y := math.Sin(dLng) * math.Cos(lat2Rad)
	x := math.Cos(lat1Rad)*math.Sin(lat2Rad) -
		math.Sin(lat1Rad)*math.Cos(lat2Rad)*math.Cos(dLng)

	bearingRad := math.Atan2(y, x)
	bearingDeg := toDegrees(bearingRad)
	return math.Mod(bearingDeg+360.0, 360.0)
}

// CompassDirection returns the 16-point cardinal compass string (e.g. "N", "NE", "SSW") for a given azimuth bearing.
func CompassDirection(bearingDeg float64) string {
	directions := []string{
		"N", "NNE", "NE", "ENE", "E", "ESE", "SE", "SSE",
		"S", "SSW", "SW", "WSW", "W", "WNW", "NW", "NNW",
	}
	normalized := math.Mod(bearingDeg+360.0, 360.0)
	idx := int(math.Floor((normalized + 11.25) / 22.5))
	return directions[idx%16]
}

// FormatDistance formats meters into human-readable e-commerce / delivery distance (e.g. "450 m", "1.2 km").
func FormatDistance(meters float64) string {
	if meters < 1000 {
		return fmt.Sprintf("%.0f m", meters)
	}
	km := meters / 1000.0
	if km < 10 {
		return fmt.Sprintf("%.1f km", km)
	}
	return fmt.Sprintf("%.0f km", km)
}
