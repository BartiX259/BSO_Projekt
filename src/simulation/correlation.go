package simulation

import "math"

// Calculates the periodic autocorrelation of a bit sequence.
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

// BitsToSignal converts a BitSequence (0s and 1s) to a slice of float32 (+1.0 for 1, -1.0 for 0).
func BitsToSignal(seq BitSequence) []float32 {
	L := seq.Len()
	signal := make([]float32, L)
	for i := 0; i < L; i++ {
		if seq.Get(i) == 1 {
			signal[i] = 1.0
		} else {
			signal[i] = -1.0
		}
	}
	return signal
}

// CalculateNormalizedCrossCorrelation calculates the normalized cross-correlation between two signals.
// Signals are expected to be slices of +1/-1 values.
// Returns a value between -1 and 1.
func CalculateNormalizedCrossCorrelation(signal1 []float32, signal2 []float32) float32 {
	if len(signal1) == 0 || len(signal1) != len(signal2) {
		panic("Signals must be non-empty and of equal length for cross-correlation.")
	}
	L := len(signal1)
	sum := float32(0.0)
	for i := 0; i < L; i++ {
		sum += signal1[i] * signal2[i]
	}
	return sum / float32(L)
}

// CalculateCorrelationSum calculates the sum part of the correlation: sum(signal1[i] * signal2[i]).
// This is used in the decoding step for CDMA.
func CalculateCorrelationSum(signal1 []float32, signal2 []float32) float32 {
	if len(signal1) == 0 || len(signal1) != len(signal2) {
		panic("Signals must be non-empty and of equal length for correlation sum.")
	}
	L := len(signal1)
	sum := float32(0.0)
	for i := 0; i < L; i++ {
		sum += signal1[i] * signal2[i]
	}
	return sum
}
