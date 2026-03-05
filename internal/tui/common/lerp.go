package common

// Lerp linearly interpolates from a toward b by factor t (0.0–1.0).
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}
