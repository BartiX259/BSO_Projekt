package simulation

func CalculateBER(originalSequence BitSequence, decodedSequence BitSequence) float32 {
	if originalSequence.length != decodedSequence.length {
		panic("Original sequence length not equal to decoded length")
	}
	errorCount := 0
	for i := range originalSequence.length {
		b1 := originalSequence.Get(i)
		b2 := decodedSequence.Get(i)
		if b1 != b2 {
			errorCount += 1
		}
	}
	return float32(errorCount) / float32(originalSequence.length)
}
