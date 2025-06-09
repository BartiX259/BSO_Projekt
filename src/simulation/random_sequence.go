package simulation

import (
	"math/rand"
	"strings"
)

// Create a random BitSequence of length N
func RandomSequence(N int) *BitSequence {
	seq := NewBitSequence(N)
	for i := range N {
		seq.Set(i, uint8(rand.Intn(2)))
	}
	return seq
}

// Create a random BitSequence of length N composed of ascii characters
func RandomText(N int) string {
	numChars := N / 8
	if numChars == 0 {
		return ""
	}
	var sb strings.Builder
	sb.Grow(numChars) // Pre-allocate memory for efficiency
	for range numChars {
		randomCharByte := byte(rand.Intn(122-97) + 97)
		sb.WriteByte(randomCharByte)
	}

	return sb.String()
}

// Convert a string into its ASCII bit sequence
func StringAsSequence(s string) *BitSequence {
	N := len(s) * 8
	seq := NewBitSequence(N)

	for i, ch := range []byte(s) {
		for j := range 8 {
			bit := (ch >> (7 - j)) & 1
			seq.Set(i*8+j, bit)
		}
	}
	return seq
}
