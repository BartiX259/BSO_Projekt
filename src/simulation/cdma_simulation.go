package simulation

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// CDMAResult holds all data for the CDMA simulation.
type CDMAResult struct {
	N                  uint
	Poly1              []uint
	Poly2              []uint
	SeedA1             uint64
	SeedA2             uint64
	SeedB1             uint64
	SeedB2             uint64
	NoiseLevel         float64
	InputTextA         string
	InputTextB         string
	SeqLengthForRandom int // Used if texts are empty, in bits

	OriginalDataSeqA *BitSequence // Original bit sequence for User A
	OriginalDataSeqB *BitSequence // Original bit sequence for User B
	EncodedDataSeqA  *BitSequence // Encoded bit sequence for User A
	EncodedDataSeqB  *BitSequence // Encoded bit sequence for User B
	DecodedDataSeqA  *BitSequence // Decoded bit sequence for User A
	DecodedDataSeqB  *BitSequence // Decoded bit sequence for User B

	GoldCodeA          *BitSequence // Actual Gold Code A object
	GoldCodeB          *BitSequence // Actual Gold Code B object
	GoldCodeAStr       string       // Full bit string of Gold Code A
	GoldCodeBStr       string       // Full bit string of Gold Code B
	CrossCorrelationAB float32      // Normalized Cross-correlation of GoldCodeA_signal and GoldCodeB_signal (zero shift)

	// Code Properties for Module 5
	AutocorrelationPeak        int     // GoldCodeLength
	MaxOffPeakAutocorrelationA float32 // Normalized
	MaxOffPeakAutocorrelationB float32 // Normalized

	CombinedSignalStr string // Truncated string of combined signal
	ReceivedSignalStr string // Truncated string of received signal (with noise)

	ReceivedSignalSegmentAStr string // NEW: Truncated string of received signal segment for User A
	ReceivedSignalSegmentBStr string // NEW: Truncated string of received signal segment for User B

	CorrelatedSignalUserAStr string // NEW: Truncated string of correlation sums for User A
	CorrelatedSignalUserBStr string // NEW: Truncated string of correlation sums for User B

	BER_A        float32
	ErrorCountA  int
	BER_B        float32
	ErrorCountB  int
	DecodedTextA string // ASCII of decoded bits for A
	DecodedTextB string // ASCII of decoded bits for B

	DataBitLengthUserA   int // Actual number of data bits for User A
	DataBitLengthUserB   int // Actual number of data bits for User B
	SimulationDataLength int // Max of DataBitLengthUserA and DataBitLengthUserB, used for simulation loops
	GoldCodeLength       int
	Timestamp            string

	TransmittedSignalAStr string // Truncated string of transmitted signal for User A
	TransmittedSignalBStr string // Truncated string of transmitted signal for User B
}

// SimulateCDMA performs the multi-user CDMA simulation using existing modules.
func SimulateCDMA(n uint, poly1 []uint, poly2 []uint,
	seedA1, seedA2 uint64, textA string,
	seedB1, seedB2 uint64, textB string,
	seqLengthForRandomBits int, noiseLevel float64) *CDMAResult {

	// Ensure different codes if seeds are identical
	if seedA1 == seedB1 && seedA2 == seedB2 {
		if seedB2 > 1 {
			seedB2--
		} else {
			seedB2++
		}
		if seedA1 == seedB1 && seedA2 == seedB2 {
			seedB1++
		}
	}

	// Generate Gold codes using existing module
	goldCodeA := GenerateGoldCode(n, poly1, seedA1, poly2, seedA2)
	goldCodeB := GenerateGoldCode(n, poly1, seedB1, poly2, seedB2)
	goldCodeLength := goldCodeA.Len()

	// Calculate code properties using existing correlation module
	signalCodeA := BitsToSignal(*goldCodeA)
	signalCodeB := BitsToSignal(*goldCodeB)
	autocorrPeak := goldCodeLength
	maxOffPeakAutoA := MaxAbsoluteOffPeak(CalculatePeriodicAutocorrelation(*goldCodeA))
	maxOffPeakAutoB := MaxAbsoluteOffPeak(CalculatePeriodicAutocorrelation(*goldCodeB))
	crossCorrAB_normalized := CalculateNormalizedCrossCorrelation(signalCodeA, signalCodeB)

	// Generate data sequences using existing module
	var dataSeqA, dataSeqB *BitSequence
	inputIsTextA := textA != ""
	inputIsTextB := textB != ""

	if inputIsTextA {
		dataSeqA = StringAsSequence(textA)
	} else {
		dataSeqA = RandomSequence(seqLengthForRandomBits)
	}
	if inputIsTextB {
		dataSeqB = StringAsSequence(textB)
	} else {
		dataSeqB = RandomSequence(seqLengthForRandomBits)
	}

	dataLenA := dataSeqA.Len()
	dataLenB := dataSeqB.Len()

	// Determine simulation length
	simulationDataLen := dataLenA
	if dataLenB > simulationDataLen {
		simulationDataLen = dataLenB
	}
	if simulationDataLen == 0 {
		simulationDataLen = 1
		if dataLenA == 0 {
			dataSeqA = RandomSequence(1)
			dataLenA = 1
		}
		if dataLenB == 0 {
			dataSeqB = RandomSequence(1)
			dataLenB = 1
		}
	}

	// Pad sequences to simulation length
	paddedDataA := NewBitSequence(simulationDataLen)
	paddedDataB := NewBitSequence(simulationDataLen)
	for i := 0; i < simulationDataLen; i++ {
		if i < dataLenA {
			paddedDataA.Set(i, dataSeqA.Get(i))
		}
		if i < dataLenB {
			paddedDataB.Set(i, dataSeqB.Get(i))
		}
	}

	// Encode data using existing encoding module
	encodedDataA := EncodeWithGold(*paddedDataA, *goldCodeA)
	encodedDataB := EncodeWithGold(*paddedDataB, *goldCodeB)

	// Create transmitted signals by spreading each data bit with the Gold code
	transmittedSignalA := make([]float32, simulationDataLen*goldCodeLength)
	transmittedSignalB := make([]float32, simulationDataLen*goldCodeLength)

	for i := 0; i < simulationDataLen; i++ {
		// Get data bit for this position
		dataBitA := float32(1)
		if paddedDataA.Get(i) == 0 {
			dataBitA = -1
		}
		dataBitB := float32(1)
		if paddedDataB.Get(i) == 0 {
			dataBitB = -1
		}

		// Spread with Gold code
		for j := 0; j < goldCodeLength; j++ {
			chipA := float32(1)
			if goldCodeA.Get(j) == 0 {
				chipA = -1
			}
			chipB := float32(1)
			if goldCodeB.Get(j) == 0 {
				chipB = -1
			}

			transmittedSignalA[i*goldCodeLength+j] = dataBitA * chipA
			transmittedSignalB[i*goldCodeLength+j] = dataBitB * chipB
		}
	}

	// Combine signals (channel transmission)
	totalSignalLength := len(transmittedSignalA)
	combinedSignal := make([]float32, totalSignalLength)
	for i := 0; i < totalSignalLength; i++ {
		combinedSignal[i] = transmittedSignalA[i] + transmittedSignalB[i]
	}

	// Add noise using similar approach to error injection
	receivedSignal := make([]float32, totalSignalLength)
	noiseRandSource := rand.NewSource(time.Now().UnixNano())
	noiseRand := rand.New(noiseRandSource)
	for i := 0; i < totalSignalLength; i++ {
		// Use Gaussian noise with standard deviation equal to noiseLevel (decimal)
		noise := noiseRand.NormFloat64() * noiseLevel // Changed: removed * 2.0
		receivedSignal[i] = combinedSignal[i] + float32(noise)
	}

	// Convert received signal back to bit sequences for decoding using correlation
	// Also get the raw correlation sums
	receivedBitsA, corrSumsA_full := signalToBitsCorrelation(receivedSignal, signalCodeA, goldCodeLength, simulationDataLen)
	receivedBitsB, corrSumsB_full := signalToBitsCorrelation(receivedSignal, signalCodeB, goldCodeLength, simulationDataLen)

	// Decoded sequences are already the final result from correlation decoding
	finalDecodedA := receivedBitsA
	finalDecodedB := receivedBitsB

	// Trim decoded sequences to original data lengths
	if finalDecodedA.Len() > dataLenA {
		trimmedDecodedA := NewBitSequence(dataLenA)
		for i := 0; i < dataLenA; i++ {
			trimmedDecodedA.Set(i, finalDecodedA.Get(i))
		}
		finalDecodedA = trimmedDecodedA
	}

	if finalDecodedB.Len() > dataLenB {
		trimmedDecodedB := NewBitSequence(dataLenB)
		for i := 0; i < dataLenB; i++ {
			trimmedDecodedB.Set(i, finalDecodedB.Get(i))
		}
		finalDecodedB = trimmedDecodedB
	}

	// Calculate BER using existing BER module
	var berA, berB float32
	var errCountA, errCountB int

	if dataLenA > 0 {
		berA = CalculateBER(*dataSeqA, *finalDecodedA)
		for i := 0; i < dataLenA; i++ {
			if dataSeqA.Get(i) != finalDecodedA.Get(i) {
				errCountA++
			}
		}
	}

	if dataLenB > 0 {
		berB = CalculateBER(*dataSeqB, *finalDecodedB)
		for i := 0; i < dataLenB; i++ {
			if dataSeqB.Get(i) != finalDecodedB.Get(i) {
				errCountB++
			}
		}
	}

	// Convert decoded bits to text if original was text
	decodedTextA := ""
	if inputIsTextA && finalDecodedA.Len() > 0 && finalDecodedA.Len()%8 == 0 {
		decodedTextA = BitsToASCII(finalDecodedA.String())
	}
	decodedTextB := ""
	if inputIsTextB && finalDecodedB.Len() > 0 && finalDecodedB.Len()%8 == 0 {
		decodedTextB = BitsToASCII(finalDecodedB.String())
	}

	displayLimit := 40
	displayLimitSignalSegment := 40   // Can be different if needed, e.g. goldCodeLength or a fixed number
	displayLimitCorrelationSums := 20 // Display fewer correlation sums as there's one per data bit

	// Populate received signal segments for each user
	// For User A
	var receivedSignalSegmentAStr string
	if dataLenA > 0 {
		// Take the segment corresponding to User A's actual data bits
		// The total length of signal part for user A is dataLenA * goldCodeLength
		endIndexA := dataLenA * goldCodeLength
		if endIndexA > len(receivedSignal) {
			endIndexA = len(receivedSignal)
		}
		if endIndexA > 0 {
			receivedSignalSegmentAStr = floatSignalToString(receivedSignal[:endIndexA], displayLimitSignalSegment)
		}
	}

	// For User B
	var receivedSignalSegmentBStr string
	if dataLenB > 0 {
		// Take the segment corresponding to User B's actual data bits
		endIndexB := dataLenB * goldCodeLength
		if endIndexB > len(receivedSignal) {
			endIndexB = len(receivedSignal)
		}
		if endIndexB > 0 {
			receivedSignalSegmentBStr = floatSignalToString(receivedSignal[:endIndexB], displayLimitSignalSegment)
		}
	}

	// Trim correlation sums to actual data lengths and convert to string
	var corrSumsA, corrSumsB []float32
	var correlatedSignalUserAStr, correlatedSignalUserBStr string

	if dataLenA > 0 && len(corrSumsA_full) >= dataLenA {
		corrSumsA = corrSumsA_full[:dataLenA]
		correlatedSignalUserAStr = floatSignalToString(corrSumsA, displayLimitCorrelationSums)
	}
	if dataLenB > 0 && len(corrSumsB_full) >= dataLenB {
		corrSumsB = corrSumsB_full[:dataLenB]
		correlatedSignalUserBStr = floatSignalToString(corrSumsB, displayLimitCorrelationSums)
	}

	return &CDMAResult{
		N:                          n,
		Poly1:                      poly1,
		Poly2:                      poly2,
		SeedA1:                     seedA1,
		SeedA2:                     seedA2,
		SeedB1:                     seedB1,
		SeedB2:                     seedB2,
		NoiseLevel:                 noiseLevel,
		InputTextA:                 textA,
		InputTextB:                 textB,
		SeqLengthForRandom:         seqLengthForRandomBits,
		OriginalDataSeqA:           dataSeqA,
		OriginalDataSeqB:           dataSeqB,
		EncodedDataSeqA:            encodedDataA,
		EncodedDataSeqB:            encodedDataB,
		DecodedDataSeqA:            finalDecodedA,
		DecodedDataSeqB:            finalDecodedB,
		GoldCodeA:                  goldCodeA,
		GoldCodeB:                  goldCodeB,
		GoldCodeAStr:               goldCodeA.String(),
		GoldCodeBStr:               goldCodeB.String(),
		CrossCorrelationAB:         crossCorrAB_normalized,
		AutocorrelationPeak:        autocorrPeak,
		MaxOffPeakAutocorrelationA: maxOffPeakAutoA,
		MaxOffPeakAutocorrelationB: maxOffPeakAutoB,
		TransmittedSignalAStr:      floatSignalToString(transmittedSignalA, displayLimit),
		TransmittedSignalBStr:      floatSignalToString(transmittedSignalB, displayLimit),
		CombinedSignalStr:          floatSignalToString(combinedSignal, displayLimit),
		ReceivedSignalStr:          floatSignalToString(receivedSignal, displayLimit),
		ReceivedSignalSegmentAStr:  receivedSignalSegmentAStr, // NEW
		ReceivedSignalSegmentBStr:  receivedSignalSegmentBStr, // NEW
		CorrelatedSignalUserAStr:   correlatedSignalUserAStr,  // NEW
		CorrelatedSignalUserBStr:   correlatedSignalUserBStr,  // NEW
		BER_A:                      berA,
		ErrorCountA:                errCountA,
		BER_B:                      berB,
		ErrorCountB:                errCountB,
		DecodedTextA:               decodedTextA,
		DecodedTextB:               decodedTextB,
		DataBitLengthUserA:         dataLenA,
		DataBitLengthUserB:         dataLenB,
		SimulationDataLength:       simulationDataLen,
		GoldCodeLength:             goldCodeLength,
		Timestamp:                  time.Now().Format(time.RFC1123),
	}
}

// signalToBitsCorrelation converts received signal back to bits using correlation with Gold code
// It now also returns a slice of the raw correlation sums.
func signalToBitsCorrelation(receivedSignal []float32, goldCodeSignal []float32, goldCodeLength int, dataBits int) (*BitSequence, []float32) {
	result := NewBitSequence(dataBits)
	correlationSums := make([]float32, dataBits) // Slice to store correlation sums

	for i := 0; i < dataBits; i++ {
		segmentStart := i * goldCodeLength
		segmentEnd := (i + 1) * goldCodeLength
		if segmentEnd > len(receivedSignal) {
			segmentEnd = len(receivedSignal)
		}
		if segmentStart >= segmentEnd {
			continue
		}

		receivedSegment := receivedSignal[segmentStart:segmentEnd]

		if len(receivedSegment) == goldCodeLength {
			corrSum := CalculateCorrelationSum(receivedSegment, goldCodeSignal)
			correlationSums[i] = corrSum // Store the sum
			// Decision based on correlation: positive = bit 1, negative = bit 0
			if corrSum > 0 {
				result.Set(i, 1)
			} else {
				result.Set(i, 0)
			}
		} else {
			// If segment is not valid (e.g., at the very end of a signal not perfectly divisible)
			// Store a default/neutral correlation sum, or handle as error if necessary
			correlationSums[i] = 0.0
		}
	}

	return result, correlationSums
}

// floatSignalToString converts a []float32 signal to a truncated string for display.
func floatSignalToString(signal []float32, limit int) string {
	var sb strings.Builder
	count := 0
	for i, val := range signal {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(fmt.Sprintf("%.2f", val))
		count++
		if limit != -1 && count >= limit {
			if len(signal) > limit {
				sb.WriteString("...")
			}
			break
		}
	}
	return sb.String()
}

// BitsToASCII converts a string of '0' and '1' to ASCII if length is a multiple of 8
func BitsToASCII(bits string) string {
	if len(bits)%8 != 0 || len(bits) == 0 {
		return "(długość bitów nie jest wielokrotnością 8)"
	}
	var sb strings.Builder
	for i := 0; i < len(bits); i += 8 {
		byteStr := bits[i : i+8]
		var bVal byte
		_, err := fmt.Sscanf(byteStr, "%b", &bVal)
		if err != nil {
			return "(błąd konwersji bitów na bajt)"
		}
		sb.WriteByte(bVal)
	}
	return sb.String()
}
