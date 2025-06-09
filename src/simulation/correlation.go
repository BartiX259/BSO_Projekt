package simulation

import "math"

// CalculatePeriodicAutocorrelation calculates the periodic autocorrelation of a bit sequence.
// Returns an array of correlation values for shifts 0 to L-1.
// Values are normalized between -1 and 1.
func CalculatePeriodicAutocorrelation(seq BitSequence) []float32 {
	L := seq.Len()
	if L == 0 {
		return []float32{}
	}
	autocorr := make([]float32, L)

	for shift := 0; shift < L; shift++ {
		sum := 0
		for i := 0; i < L; i++ {
			b1 := seq.Get(i)
			b2 := seq.Get((i + shift) % L) // Apply shift with periodic boundary
			if b1 == b2 {
				sum++ // Agreement
			} else {
				sum-- // Disagreement
			}
		}
		autocorr[shift] = float32(sum) / float32(L)
	}
	return autocorr
}

// Helper to find max absolute value in a slice, ignoring the first element (for off-peak)
func MaxAbsoluteOffPeak(values []float32) float32 {
	if len(values) <= 1 {
		return 0.0
	}
	maxVal := float32(0.0)
	for i := 1; i < len(values); i++ { // Start from index 1
		absVal := float32(math.Abs(float64(values[i])))
		if absVal > maxVal {
			maxVal = absVal
		}
	}
	return maxVal
}
