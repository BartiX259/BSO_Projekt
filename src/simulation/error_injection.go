package simulation

import (
	"math/rand"
)

// Introduces errors to a bit sequence based on specified parameters
func AddErrors(sequence *BitSequence, errorRate float64, errorType string) (*BitSequence, int) {
	if errorRate <= 0 {
		// Return copy of original sequence with no errors
		result := NewBitSequence(sequence.Len())
		for i := range sequence.Len() {
			result.Set(i, sequence.Get(i))
		}
		return result, 0
	}

	corrupted := NewBitSequence(sequence.Len())
	errorsIntroduced := 0

	// Copy original sequence
	for i := range sequence.Len() {
		corrupted.Set(i, sequence.Get(i))
	}

	if errorType == "random" {
		// Random errors
		for i := range sequence.Len() {
			if rand.Float64() < (errorRate / 100.0) {
				// Flip the bit
				corrupted.Set(i, 1-corrupted.Get(i))
				errorsIntroduced++
			}
		}
	} else if errorType == "burst" {
		// Burst errors - simplified implementation
		burstLength := 3
		numBursts := int(float64(sequence.Len()) * errorRate / 100.0 / float64(burstLength))
		
		for range numBursts {
			startPos := rand.Intn(sequence.Len() - burstLength)
			for i := 0; i < burstLength && startPos+i < sequence.Len(); i++ {
				corrupted.Set(startPos+i, 1-corrupted.Get(startPos+i))
				errorsIntroduced++
			}
		}
	}

	return corrupted, errorsIntroduced
}
