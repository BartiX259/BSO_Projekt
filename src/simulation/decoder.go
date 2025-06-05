package simulation

func DecodeWithGold(dataSequence BitSequence, goldCode BitSequence) *BitSequence {
	if dataSequence.length > goldCode.length {
		panic("Data sequence length higher than gold code length.")
	}
	decodedSequence := NewBitSequence(dataSequence.length)
	for i := range dataSequence.length {
		nextBit := dataSequence.Get(i) ^ goldCode.Get(i)
		decodedSequence.Set(i, nextBit)
	}
	return decodedSequence
}
