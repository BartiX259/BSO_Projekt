package src

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/BartiX259/BSO_Projekt/src/simulation"
)

const (
	generalSimOutputDir  = "simulation_data"
	cdmaSimOutputDir     = "cdma_simulation_data"
	generalSimFilePrefix = "simulation_results_"
	cdmaSimFilePrefix    = "cdma_simulation_results_"
	fileSuffix           = ".txt"
	maxFilesToKeep       = 5
)

func init() {
	ensureDirExists(generalSimOutputDir)
	ensureDirExists(cdmaSimOutputDir)
}

func ensureDirExists(dirPath string) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.Printf("Creating directory: %s", dirPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Fatalf("Failed to create directory '%s': %v", dirPath, err)
		}
	}
}

// --- General Simulation Results Formatting and Saving ---

func FormatSimulationResultsToText(results *SimulationResults) string {
	if results == nil {
		return "No general simulation data available.\n"
	}
	var sb strings.Builder
	ts := results.Timestamp
	if ts == "" {
		ts = "N/A"
	}

	sb.WriteString(fmt.Sprintf("Simulation Results - Timestamp: %s\n", ts))
	sb.WriteString("==================================================\n\n")
	sb.WriteString("Input Parameters:\n")
	sb.WriteString(fmt.Sprintf("  Input Text: %s\n", results.InputText))
	sb.WriteString(fmt.Sprintf("  Gold Code N: %d\n", results.GoldN))
	sb.WriteString(fmt.Sprintf("  Gold Taps1: %v\n", results.GoldTaps1))
	sb.WriteString(fmt.Sprintf("  Gold Taps2: %v\n", results.GoldTaps2))
	sb.WriteString(fmt.Sprintf("  Decoder Type: %s\n", results.DecoderType))
	sb.WriteString(fmt.Sprintf("  Error Type: %s\n", results.ErrorType))
	sb.WriteString(fmt.Sprintf("  Error Rate: %.2f%%\n", results.ErrorRate))
	sb.WriteString("\nGenerated/Processed Sequences:\n")
	if results.Original != nil {
		sb.WriteString(fmt.Sprintf("  Original (len %d): %s\n", results.Original.Len(), results.Original.String()))
		if results.InputText != "" {
			sb.WriteString(fmt.Sprintf("  Original ASCII: %s\n", bitsToASCII(results.Original.String())))
		}
	} else {
		sb.WriteString("  Original Sequence: Not available\n")
	}
	if results.GoldCode != nil {
		sb.WriteString(fmt.Sprintf("  Gold Code (len %d): %s\n", results.GoldCode.Len(), results.GoldCode.String()))
	} else {
		sb.WriteString("  Gold Code: Not available\n")
	}
	if results.Encoded != nil {
		sb.WriteString(fmt.Sprintf("  Encoded (len %d): %s\n", results.Encoded.Len(), results.Encoded.String()))
	} else {
		sb.WriteString("  Encoded Sequence: Not available / Module disabled\n")
	}
	if results.Corrupted != nil {
		sb.WriteString(fmt.Sprintf("  Corrupted (len %d): %s\n", results.Corrupted.Len(), results.Corrupted.String()))
		sb.WriteString(fmt.Sprintf("  Errors Introduced: %d\n", results.ErrorsIntroduced))
	} else {
		sb.WriteString("  Corrupted Sequence: Not available / Error module disabled\n")
	}
	if results.Decoded != nil {
		sb.WriteString(fmt.Sprintf("  Decoded (len %d): %s\n", results.Decoded.Len(), results.Decoded.String()))
		if results.InputText != "" {
			sb.WriteString(fmt.Sprintf("  Decoded ASCII: %s\n", bitsToASCII(results.Decoded.String())))
		}
	} else {
		sb.WriteString("  Decoded Sequence: Not available / Decoder module disabled\n")
	}
	sb.WriteString("\nAnalysis Results:\n")
	if results.Original != nil && results.Decoded != nil {
		sb.WriteString(fmt.Sprintf("  BER: %.4f (%.2f%%)\n", results.BER, results.BER*100))
		sb.WriteString(fmt.Sprintf("  Error Count (vs Original): %d / %d bits\n", results.ErrorCount, results.Original.Len()))
	} else {
		sb.WriteString("  BER: Not calculated / Relevant modules disabled\n")
	}
	sb.WriteString("\nAutocorrelation (Max Absolute Off-Peak):\n")
	if results.OriginalAutocorr != 0 || results.EncodedAutocorr != 0 || results.CorruptedAutocorr != 0 || (results.Original != nil) {
		sb.WriteString(fmt.Sprintf("  Original: %.4f\n", results.OriginalAutocorr))
		sb.WriteString(fmt.Sprintf("  Encoded: %.4f\n", results.EncodedAutocorr))
		sb.WriteString(fmt.Sprintf("  Corrupted: %.4f\n", results.CorruptedAutocorr))
	} else {
		sb.WriteString("  Autocorrelation analysis not performed or results are zero.\n")
	}
	sb.WriteString("\n==================================================\nEnd of Report\n")
	return sb.String()
}

func SaveSimulationResultsToFile(results *SimulationResults) (string, error) {
	if results == nil {
		return "", fmt.Errorf("cannot save nil SimulationResults")
	}
	content := FormatSimulationResultsToText(results)
	filePath, err := saveContentToFile(generalSimOutputDir, generalSimFilePrefix, results.Timestamp, content)
	if err != nil {
		return "", err
	}
	cleanupOldFiles(generalSimOutputDir, generalSimFilePrefix, fileSuffix, maxFilesToKeep)
	return filePath, nil
}

// --- CDMA Simulation Results Formatting and Saving ---

func FormatCDMAResultsToText(results *simulation.CDMAResult) string {
	if results == nil {
		return "No CDMA simulation data available.\n"
	}
	var sb strings.Builder
	ts := results.Timestamp
	if ts == "" {
		ts = "N/A"
	}

	sb.WriteString(fmt.Sprintf("CDMA Simulation Results - Timestamp: %s\n", ts))
	sb.WriteString("======================================================\n\n")
	sb.WriteString("Input Parameters:\n")
	sb.WriteString(fmt.Sprintf("  Gold Code N: %d\n", results.N))
	sb.WriteString(fmt.Sprintf("  Poly1 Taps: %v, Poly2 Taps: %v\n", results.Poly1, results.Poly2))
	sb.WriteString(fmt.Sprintf("  User A Seeds (L1/L2): 0x%X / 0x%X\n", results.SeedA1, results.SeedA2))
	sb.WriteString(fmt.Sprintf("  User B Seeds (L1/L2): 0x%X / 0x%X\n", results.SeedB1, results.SeedB2))
	sb.WriteString(fmt.Sprintf("  Noise Level: %.4f\n", results.NoiseLevel))
	sb.WriteString(fmt.Sprintf("  Input Text A: \"%s\", Input Text B: \"%s\"\n", results.InputTextA, results.InputTextB))
	if results.InputTextA == "" && results.InputTextB == "" {
		sb.WriteString(fmt.Sprintf("  Random Seq Length: %d bits\n", results.SeqLengthForRandom))
	}
	sb.WriteString("\nData Lengths & Codes:\n")
	sb.WriteString(fmt.Sprintf("  User A Data Bits: %d, User B Data Bits: %d\n", results.DataBitLengthUserA, results.DataBitLengthUserB))
	sb.WriteString(fmt.Sprintf("  Gold Code Length: %d\n", results.GoldCodeLength))
	sb.WriteString(fmt.Sprintf("  User A Gold Code: %s\n", results.GoldCodeAStr))
	sb.WriteString(fmt.Sprintf("  User B Gold Code: %s\n", results.GoldCodeBStr))
	sb.WriteString("\nCode Properties:\n")
	sb.WriteString(fmt.Sprintf("  Autocorr Peak: %d, Max Off-Peak A: %.4f, Max Off-Peak B: %.4f\n", results.AutocorrelationPeak, results.MaxOffPeakAutocorrelationA, results.MaxOffPeakAutocorrelationB))
	sb.WriteString(fmt.Sprintf("  Cross-Correlation (A vs B): %.4f\n", results.CrossCorrelationAB))
	sb.WriteString("\nUser A Path:\n")
	if results.OriginalDataSeqA != nil {
		sb.WriteString(fmt.Sprintf("  Original A: %s\n", results.OriginalDataSeqA.String()))
	}
	if results.EncodedDataSeqA != nil {
		sb.WriteString(fmt.Sprintf("  Encoded A: %s\n", results.EncodedDataSeqA.String()))
	}
	sb.WriteString(fmt.Sprintf("  Transmitted A (trunc): %s\n", results.TransmittedSignalAStr))
	sb.WriteString("\nUser B Path:\n")
	if results.OriginalDataSeqB != nil {
		sb.WriteString(fmt.Sprintf("  Original B: %s\n", results.OriginalDataSeqB.String()))
	}
	if results.EncodedDataSeqB != nil {
		sb.WriteString(fmt.Sprintf("  Encoded B: %s\n", results.EncodedDataSeqB.String()))
	}
	sb.WriteString(fmt.Sprintf("  Transmitted B (trunc): %s\n", results.TransmittedSignalBStr))
	sb.WriteString("\nChannel & Reception:\n")
	sb.WriteString(fmt.Sprintf("  Combined (trunc): %s\n", results.CombinedSignalStr))
	sb.WriteString(fmt.Sprintf("  Received (trunc): %s\n", results.ReceivedSignalStr))
	sb.WriteString(fmt.Sprintf("  Rx Segment A (trunc): %s, Rx Segment B (trunc): %s\n", results.ReceivedSignalSegmentAStr, results.ReceivedSignalSegmentBStr))
	sb.WriteString("\nUser A Decoding:\n")
	sb.WriteString(fmt.Sprintf("  Correlated A (trunc): %s\n", results.CorrelatedSignalUserAStr))
	if results.DecodedDataSeqA != nil {
		sb.WriteString(fmt.Sprintf("  Decoded A: %s\n", results.DecodedDataSeqA.String()))
		sb.WriteString(fmt.Sprintf("  Decoded Text A: \"%s\"\n", results.DecodedTextA))
	}
	sb.WriteString(fmt.Sprintf("  BER A: %.2f%%, Errors A: %d/%d\n", results.BER_A*100, results.ErrorCountA, results.DataBitLengthUserA))
	sb.WriteString("\nUser B Decoding:\n")
	sb.WriteString(fmt.Sprintf("  Correlated B (trunc): %s\n", results.CorrelatedSignalUserBStr))
	if results.DecodedDataSeqB != nil {
		sb.WriteString(fmt.Sprintf("  Decoded B: %s\n", results.DecodedDataSeqB.String()))
		sb.WriteString(fmt.Sprintf("  Decoded Text B: \"%s\"\n", results.DecodedTextB))
	}
	sb.WriteString(fmt.Sprintf("  BER B: %.2f%%, Errors B: %d/%d\n", results.BER_B*100, results.ErrorCountB, results.DataBitLengthUserB))
	sb.WriteString("\n======================================================\nEnd of CDMA Report\n")
	return sb.String()
}

func SaveCDMAResultsToFile(results *simulation.CDMAResult) (string, error) {
	if results == nil {
		return "", fmt.Errorf("cannot save nil CDMAResult")
	}
	content := FormatCDMAResultsToText(results)
	filePath, err := saveContentToFile(cdmaSimOutputDir, cdmaSimFilePrefix, results.Timestamp, content)
	if err != nil {
		return "", err
	}
	cleanupOldFiles(cdmaSimOutputDir, cdmaSimFilePrefix, fileSuffix, maxFilesToKeep)
	return filePath, nil
}

// --- Shared Helper Functions ---

func saveContentToFile(dir, prefix, timestampStr, content string) (string, error) {
	ensureDirExists(dir) // Ensure directory exists just in case

	var t time.Time
	var errParse error
	if timestampStr != "" {
		t, errParse = time.Parse(time.RFC1123, timestampStr)
		if errParse != nil {
			log.Printf("Warning: Could not parse timestamp ('%s') for filename, using current time: %v", timestampStr, errParse)
			t = time.Now()
		}
	} else {
		log.Println("Warning: Timestamp is empty, using current time for filename.")
		t = time.Now()
	}

	filename := fmt.Sprintf("%s%s%s", prefix, t.Format("20060102_150405.000"), fileSuffix)
	filePath := filepath.Join(dir, filename)

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Printf("Error writing to file '%s': %v", filePath, err)
		return "", err
	}
	log.Printf("Results saved to server at: %s", filePath)
	return filePath, nil
}

func cleanupOldFiles(dir, prefix, suffix string, maxFiles int) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("Error reading directory '%s' for cleanup: %v", dir, err)
		return
	}

	var eligibleFiles []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), suffix) {
			eligibleFiles = append(eligibleFiles, entry)
		}
	}

	if len(eligibleFiles) <= maxFiles {
		return
	}

	// Sort by filename (ascending, so oldest are first, assuming YYYYMMDD_HHMMSS.mmm format)
	sort.Slice(eligibleFiles, func(i, j int) bool {
		return eligibleFiles[i].Name() < eligibleFiles[j].Name()
	})

	filesToDeleteCount := len(eligibleFiles) - maxFiles
	for i := 0; i < filesToDeleteCount; i++ {
		filePathToDelete := filepath.Join(dir, eligibleFiles[i].Name())
		if err := os.Remove(filePathToDelete); err != nil {
			log.Printf("Error deleting old file '%s': %v", filePathToDelete, err)
		} else {
			log.Printf("Deleted old file: %s", filePathToDelete)
		}
	}
}