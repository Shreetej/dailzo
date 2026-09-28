package utils

import (
	"crypto/rand"
	"math"
	"math/big"
	"strings"
)

// SplitCSV turns a comma-separated id list (as stored in the users table's
// favourite_restaurants/favourite_foods TEXT columns) into a clean []string
// for JSON responses — never nil, and never containing the empty-string
// entries strings.Split produces for an empty or trailing-comma value.
func SplitCSV(s *string) []string {
	out := []string{}
	if s == nil || *s == "" {
		return out
	}
	for _, part := range strings.Split(*s, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// GenerateOTP generates a random 6-digit OTP
func GenerateOTP() string {
	numbers := "0123456789"
	otp := make([]byte, 6)
	for i := range otp {
		randomIndex, _ := rand.Int(rand.Reader, big.NewInt(int64(len(numbers))))
		otp[i] = numbers[randomIndex.Int64()]
	}
	return string(otp)
}

// haversine calculates the distance between two points (latitude, longitude) on Earth.
func GetDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371 // Earth's radius in kilometers (use 3958.8 for miles)

	// Convert degrees to radians
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	// Haversine formula
	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(deltaLon/2)*math.Sin(deltaLon/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	// Distance in kilometers
	distance := earthRadius * c

	return distance
}

func GetBoundingBox(lat, lon, radius float64) (float64, float64, float64, float64) {
	const earthRadius = 6371.0 // Earth's radius in km

	// Convert radius from km to radians
	latDelta := radius / earthRadius
	lonDelta := radius / (earthRadius * math.Cos(lat*math.Pi/180))

	// Calculate min and max latitude and longitude
	minLat := lat - latDelta*(180/math.Pi)
	maxLat := lat + latDelta*(180/math.Pi)
	minLon := lon - lonDelta*(180/math.Pi)
	maxLon := lon + lonDelta*(180/math.Pi)

	return minLat, maxLat, minLon, maxLon
}
