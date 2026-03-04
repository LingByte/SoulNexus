package utils

import "math"

// Calculate distance between two points
func GetDistance(lon1, lat1, lon2, lat2 float64) float64 {
	rad := (math.Pi / 180.0)
	r := 6371000.0 // Earth radius
	x := (lon2 - lon1) * rad * math.Cos((lat1+lat2)/2*rad)
	y := (lat2 - lat1) * rad
	return math.Sqrt(x*x+y*y) * r
}
