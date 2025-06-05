package simulation

func EncodeWithGold(dataSequence BitSequence, goldCode BitSequence) *BitSequence {
	if dataSequence.length > goldCode.length {
		panic("Data sequence length higher than gold code length.")
	}
	encodedSequence := NewBitSequence(dataSequence.length)
	for i := range dataSequence.length {
		nextBit := dataSequence.Get(i) ^ goldCode.Get(i)
		encodedSequence.Set(i, nextBit)
	}
	return encodedSequence
}

func GenerateGoldCode(n uint, poly1 []uint, seed1 uint64, poly2 []uint, seed2 uint64) *BitSequence {
	codeLength := pow2(n) - 1
	lfsr1 := NewLFSR(seed1, poly1, n)
	lfsr2 := NewLFSR(seed2, poly2, n)
	goldCode := NewBitSequence(codeLength)
	for i := range codeLength {
		bit1 := lfsr1.Shift()
		bit2 := lfsr2.Shift()
		goldBit := bit1 ^ bit2
		goldCode.Set(i, goldBit)
	}
	return goldCode
}

func pow2(n uint) int {
	res := 1
	for range n {
		res *= 2
	}
	return res
}
