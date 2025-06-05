package simulation

import (
	"math/rand"
)

// Create a random BitSequence of length N
func RandomSequence(N int) *BitSequence {
	seq := NewBitSequence(N)
	for i := range N {
		seq.Set(i, uint8(rand.Intn(2)))
	}
	return seq
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
