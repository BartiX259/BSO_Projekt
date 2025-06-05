package simulation

// Generic N-bit Linear Feedback Shift Register
type LFSR struct {
	state   uint64   // Holds the current state (max 64 bits)
	taps    []uint   // Tap positions (0-based from LSB)
	n       uint     // Register width in bits (≤ 64)
	initial uint64   // Keep the initial state for reset, if needed
}

// Initialize a new N-bit LFSR
func NewLFSR(seed uint64, taps []uint, n uint) *LFSR {
	if n == 0 || n > 64 {
		panic("LFSR size must be between 1 and 64 bits")
	}
	if seed == 0 || seed >= (1 << n) {
		panic("Seed must be non-zero and fit in N bits")
	}
	return &LFSR{
		state:   seed,
		taps:    taps,
		n:       n,
		initial: seed,
	}
}

// Advance the LFSR by one bit and return the feedback
func (l *LFSR) Shift() uint8 {
	feedback := l.Feedback()
	l.state = ((l.state << 1) | uint64(feedback)) & ((1 << l.n) - 1)
	return feedback
}

// Get the current feedback bit of the LFSR
func (l *LFSR) Feedback() uint8 {
	var feedback uint8
	for _, t := range l.taps {
		if t >= l.n {
			panic("Tap position exceeds LFSR width")
		}
		feedback ^= uint8((l.state >> t) & 1)
	}
	return feedback
}
