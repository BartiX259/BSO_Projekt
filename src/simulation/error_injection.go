package simulation

import (
	"math/rand"
	"time"
)

// AddErrors introduces errors to a bit sequence based on specified parameters
func AddErrors(sequence *BitSequence, errorRate float64, errorType string) (*BitSequence, int) {
	if errorRate <= 0 {
		// Return copy of original sequence with no errors
		result := NewBitSequence(sequence.Len())
		for i := 0; i < sequence.Len(); i++ {
			result.Set(i, sequence.Get(i))
		}
		return result, 0
	}

	rand.Seed(time.Now().UnixNano())
	corrupted := NewBitSequence(sequence.Len())
	errorsIntroduced := 0

	// Copy original sequence
	for i := 0; i < sequence.Len(); i++ {
		corrupted.Set(i, sequence.Get(i))
	}

	if errorType == "random" {
		// Random errors
		for i := 0; i < sequence.Len(); i++ {
			if rand.Float64() < (errorRate / 100.0) {
				// Flip the bit
				corrupted.Set(i, 1-corrupted.Get(i))
				errorsIntroduced++
			}
		}
	} else if errorType == "burst" {
		// Burst errors - simplified implementation
		burstLength := 3 // Fixed burst length for now
		numBursts := int(float64(sequence.Len()) * errorRate / 100.0 / float64(burstLength))
		
		for burst := 0; burst < numBursts; burst++ {
			startPos := rand.Intn(sequence.Len() - burstLength)
			for i := 0; i < burstLength && startPos+i < sequence.Len(); i++ {
				corrupted.Set(startPos+i, 1-corrupted.Get(startPos+i))
				errorsIntroduced++
			}
		}
	}

	return corrupted, errorsIntroduced
}
