package simulation

// Stores a sequence of bits up to any length N
type BitSequence struct {
	bits  []uint64
	length int
}

// Create a new BitSequence of the given length, all bits initialized to 0
func NewBitSequence(length int) *BitSequence {
	if length < 1 {
		panic("BitSequence length must be positive")
	}
	words := (length + 63) / 64
	return &BitSequence{
		bits:  make([]uint64, words),
		length: length,
	}
}

// Return the bit at the given position
func (b *BitSequence) Get(pos int) uint8 {
	if pos < 0 || pos >= b.length {
		panic("Bit index out of range")
	}
	word := pos / 64
	bit := pos % 64
	return uint8((b.bits[word] >> bit) & 1)
}

// Set the bit at the given position to 0 or 1
func (b *BitSequence) Set(pos int, value uint8) {
	if pos < 0 || pos >= b.length {
		panic("Bit index out of range")
	}
	word := pos / 64
	bit := pos % 64
	if value != 0 {
		b.bits[word] |= (1 << bit)
	} else {
		b.bits[word] &^= (1 << bit)
	}
}

// Return the total number of bits in the sequence
func (b *BitSequence) Len() int {
	return b.length
}

// Debug: print all bits as a string
func (b *BitSequence) String() string {
	str := ""
	for i := range b.length {
		if b.Get(i) == 1 {
			str += "1"
		} else {
			str += "0"
		}
	}
	return str
}
