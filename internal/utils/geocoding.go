package utils

import (
	"strings"
)

// CityCoords maps normalized lowercase Indian cities/districts to centroid lat/lng.
var cityCoords = map[string][2]float64{
	"noida":         {28.5828, 77.3110},
	"greater noida": {28.4744, 77.5040},
	"delhi":         {28.6139, 77.2090},
	"new delhi":     {28.6139, 77.2090},
	"gurgaon":       {28.4595, 77.0266},
	"gurugram":      {28.4595, 77.0266},
	"ghaziabad":     {28.6692, 77.4538},
	"faridabad":     {28.4089, 77.3178},
	"darbhanga":     {26.1542, 85.8918},
	"patna":         {25.6126, 85.1444},
	"muzaffarpur":   {26.1209, 85.3647},
	"samastipur":    {25.8629, 85.7811},
	"madhubani":     {26.3533, 86.0718},
	"bhagalpur":     {25.2425, 86.9842},
	"gaya":          {24.7914, 85.0002},
	"mumbai":        {19.0760, 72.8777},
	"pune":          {18.5204, 73.8567},
	"bangalore":     {12.9716, 77.5946},
	"bengaluru":     {12.9716, 77.5946},
	"hyderabad":     {17.3850, 78.4867},
	"kolkata":       {22.5726, 88.3639},
	"chennai":       {13.0827, 80.2707},
	"jaipur":        {26.9124, 75.7873},
	"lucknow":       {26.8467, 80.9462},
	"kanpur":        {26.4499, 80.3319},
	"varanasi":      {25.3176, 82.9739},
	"ahmedabad":     {23.0225, 72.5714},
	"surat":         {21.1702, 72.8311},
	"chandigarh":    {30.7333, 76.7794},
	"indore":        {22.7196, 75.8577},
	"bhopal":        {23.2599, 77.4126},
	"ranchi":        {23.3441, 85.3096},
}

// ResolveCityCoordinates resolves latitude and longitude based on Indian city or pincode.
func ResolveCityCoordinates(city, pincode string) (*float64, *float64) {
	c := strings.ToLower(strings.TrimSpace(city))
	for cityName, coords := range cityCoords {
		if c != "" && (strings.Contains(c, cityName) || strings.Contains(cityName, c)) {
			lat, lng := coords[0], coords[1]
			return &lat, &lng
		}
	}

	p := strings.TrimSpace(pincode)
	if len(p) >= 3 {
		switch {
		case strings.HasPrefix(p, "11"): // Delhi
			lat, lng := 28.6139, 77.2090
			return &lat, &lng
		case strings.HasPrefix(p, "201"): // Noida / Ghaziabad
			lat, lng := 28.5828, 77.3110
			return &lat, &lng
		case strings.HasPrefix(p, "122"): // Gurgaon
			lat, lng := 28.4595, 77.0266
			return &lat, &lng
		case strings.HasPrefix(p, "846"): // Darbhanga
			lat, lng := 26.1542, 85.8918
			return &lat, &lng
		case strings.HasPrefix(p, "800"): // Patna
			lat, lng := 25.6126, 85.1444
			return &lat, &lng
		case strings.HasPrefix(p, "560"): // Bengaluru
			lat, lng := 12.9716, 77.5946
			return &lat, &lng
		case strings.HasPrefix(p, "400"): // Mumbai
			lat, lng := 19.0760, 72.8777
			return &lat, &lng
		}
	}

	return nil, nil
}
