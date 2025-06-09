package simulation

import (
	"fmt"
	"math/rand"
	"strings"
	"time"
)

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
	SeqLengthForRandom int

	OriginalDataSeqA *BitSequence
	OriginalDataSeqB *BitSequence
	EncodedDataSeqA  *BitSequence
	EncodedDataSeqB  *BitSequence
	DecodedDataSeqA  *BitSequence
	DecodedDataSeqB  *BitSequence

	GoldCodeA          *BitSequence
	GoldCodeB          *BitSequence
	GoldCodeAStr       string
	GoldCodeBStr       string
	CrossCorrelationAB float32

	AutocorrelationPeak        int
	MaxOffPeakAutocorrelationA float32
	MaxOffPeakAutocorrelationB float32

	CombinedSignalStr string
	ReceivedSignalStr string

	ReceivedSignalSegmentAStr string
	ReceivedSignalSegmentBStr string

	CorrelatedSignalUserAStr string
	CorrelatedSignalUserBStr string

	BER_A        float32
	ErrorCountA  int
	BER_B        float32
	ErrorCountB  int
	DecodedTextA string
	DecodedTextB string

	DataBitLengthUserA   int
	DataBitLengthUserB   int
	SimulationDataLength int
	GoldCodeLength       int
	Timestamp            string

	TransmittedSignalAStr       string
	TransmittedSignalBStr       string
	FullTransmittedSignalLength int
}

func SimulateCDMA(n uint, poly1 []uint, poly2 []uint,
	seedA1, seedA2 uint64, textA string,
	seedB1, seedB2 uint64, textB string,
	seqLengthForRandomBits int, noiseLevel float64) *CDMAResult {

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

	goldCodeA := GenerateGoldCode(n, poly1, seedA1, poly2, seedA2)
	goldCodeB := GenerateGoldCode(n, poly1, seedB1, poly2, seedB2)
	goldCodeLength := goldCodeA.Len()

	signalCodeA := BitsToSignal(*goldCodeA)
	signalCodeB := BitsToSignal(*goldCodeB)
	autocorrPeak := goldCodeLength
	maxOffPeakAutoA := MaxAbsoluteOffPeak(CalculatePeriodicAutocorrelation(*goldCodeA))
	maxOffPeakAutoB := MaxAbsoluteOffPeak(CalculatePeriodicAutocorrelation(*goldCodeB))
	crossCorrAB_normalized := CalculateNormalizedCrossCorrelation(signalCodeA, signalCodeB)

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

	encodedDataA := EncodeWithGold(*paddedDataA, *goldCodeA)
	encodedDataB := EncodeWithGold(*paddedDataB, *goldCodeB)

	transmittedSignalA := make([]float32, simulationDataLen*goldCodeLength)
	transmittedSignalB := make([]float32, simulationDataLen*goldCodeLength)

	for i := 0; i < simulationDataLen; i++ {
		dataBitA := float32(1)
		if paddedDataA.Get(i) == 0 {
			dataBitA = -1
		}
		dataBitB := float32(1)
		if paddedDataB.Get(i) == 0 {
			dataBitB = -1
		}

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

	totalSignalLength := len(transmittedSignalA)
	combinedSignal := make([]float32, totalSignalLength)
	for i := 0; i < totalSignalLength; i++ {
		combinedSignal[i] = transmittedSignalA[i] + transmittedSignalB[i]
	}

	receivedSignal := make([]float32, totalSignalLength)
	noiseRandSource := rand.NewSource(time.Now().UnixNano())
	noiseRand := rand.New(noiseRandSource)
	for i := 0; i < totalSignalLength; i++ {
		noise := noiseRand.NormFloat64() * noiseLevel
		receivedSignal[i] = combinedSignal[i] + float32(noise)
	}

	receivedBitsA, corrSumsA_full := signalToBitsCorrelation(receivedSignal, signalCodeA, goldCodeLength, simulationDataLen)
	receivedBitsB, corrSumsB_full := signalToBitsCorrelation(receivedSignal, signalCodeB, goldCodeLength, simulationDataLen)

	finalDecodedA := receivedBitsA
	finalDecodedB := receivedBitsB

	if finalDecodedA.Len() > dataLenA {
		trimmedDecodedA := NewBitSequence(dataLenA)
		for i := range dataLenA {
			trimmedDecodedA.Set(i, finalDecodedA.Get(i))
		}
		finalDecodedA = trimmedDecodedA
	}

	if finalDecodedB.Len() > dataLenB {
		trimmedDecodedB := NewBitSequence(dataLenB)
		for i := range dataLenB {
			trimmedDecodedB.Set(i, finalDecodedB.Get(i))
		}
		finalDecodedB = trimmedDecodedB
	}

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

	decodedTextA := ""
	if inputIsTextA && finalDecodedA.Len() > 0 && finalDecodedA.Len()%8 == 0 {
		decodedTextA = BitsToASCII(finalDecodedA.String())
	}
	decodedTextB := ""
	if inputIsTextB && finalDecodedB.Len() > 0 && finalDecodedB.Len()%8 == 0 {
		decodedTextB = BitsToASCII(finalDecodedB.String())
	}

	displayLimit := 40
	displayLimitSignalSegment := 40
	displayLimitCorrelationSums := 20

	var receivedSignalSegmentAStr string
	if dataLenA > 0 {
		endIndexA := dataLenA * goldCodeLength
		if endIndexA > len(receivedSignal) {
			endIndexA = len(receivedSignal)
		}
		if endIndexA > 0 {
			receivedSignalSegmentAStr = floatSignalToString(receivedSignal[:endIndexA], displayLimitSignalSegment)
		}
	}

	var receivedSignalSegmentBStr string
	if dataLenB > 0 {
		endIndexB := dataLenB * goldCodeLength
		if endIndexB > len(receivedSignal) {
			endIndexB = len(receivedSignal)
		}
		if endIndexB > 0 {
			receivedSignalSegmentBStr = floatSignalToString(receivedSignal[:endIndexB], displayLimitSignalSegment)
		}
	}

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
		N:                           n,
		Poly1:                       poly1,
		Poly2:                       poly2,
		SeedA1:                      seedA1,
		SeedA2:                      seedA2,
		SeedB1:                      seedB1,
		SeedB2:                      seedB2,
		NoiseLevel:                  noiseLevel,
		InputTextA:                  textA,
		InputTextB:                  textB,
		SeqLengthForRandom:          seqLengthForRandomBits,
		OriginalDataSeqA:            dataSeqA,
		OriginalDataSeqB:            dataSeqB,
		EncodedDataSeqA:             encodedDataA,
		EncodedDataSeqB:             encodedDataB,
		DecodedDataSeqA:             finalDecodedA,
		DecodedDataSeqB:             finalDecodedB,
		GoldCodeA:                   goldCodeA,
		GoldCodeB:                   goldCodeB,
		GoldCodeAStr:                goldCodeA.String(),
		GoldCodeBStr:                goldCodeB.String(),
		CrossCorrelationAB:          crossCorrAB_normalized,
		AutocorrelationPeak:         autocorrPeak,
		MaxOffPeakAutocorrelationA:  maxOffPeakAutoA,
		MaxOffPeakAutocorrelationB:  maxOffPeakAutoB,
		TransmittedSignalAStr:       floatSignalToString(transmittedSignalA, displayLimit),
		TransmittedSignalBStr:       floatSignalToString(transmittedSignalB, displayLimit),
		CombinedSignalStr:           floatSignalToString(combinedSignal, displayLimit),
		ReceivedSignalStr:           floatSignalToString(receivedSignal, displayLimit),
		ReceivedSignalSegmentAStr:   receivedSignalSegmentAStr,
		ReceivedSignalSegmentBStr:   receivedSignalSegmentBStr,
		CorrelatedSignalUserAStr:    correlatedSignalUserAStr,
		CorrelatedSignalUserBStr:    correlatedSignalUserBStr,
		BER_A:                       berA,
		ErrorCountA:                 errCountA,
		BER_B:                       berB,
		ErrorCountB:                 errCountB,
		DecodedTextA:                decodedTextA,
		DecodedTextB:                decodedTextB,
		DataBitLengthUserA:          dataLenA,
		DataBitLengthUserB:          dataLenB,
		SimulationDataLength:        simulationDataLen,
		GoldCodeLength:              goldCodeLength,
		Timestamp:                   time.Now().Format(time.RFC1123),
		FullTransmittedSignalLength: simulationDataLen * goldCodeLength,
	}
}

func signalToBitsCorrelation(receivedSignal []float32, goldCodeSignal []float32, goldCodeLength int, dataBits int) (*BitSequence, []float32) {
	result := NewBitSequence(dataBits)
	correlationSums := make([]float32, dataBits)

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
			correlationSums[i] = corrSum
			if corrSum > 0 {
				result.Set(i, 1)
			} else {
				result.Set(i, 0)
			}
		} else {
			correlationSums[i] = 0.0
		}
	}

	return result, correlationSums
}

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
