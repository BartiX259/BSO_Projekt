package src

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const outputDir = "simulation_data" // Directory to store result files
const maxResultFiles = 5

// init function to ensure directory exists at package initialization for this file's scope
func init() {
	ensureOutputDirExists()
}

// ensureOutputDirExists creates the output directory if it doesn't exist.
func ensureOutputDirExists() {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		log.Printf("Creating output directory: %s", outputDir)
		err = os.MkdirAll(outputDir, 0755) // Create directory with rwxr-xr-x permissions
		if err != nil {
			log.Fatalf("Failed to create output directory '%s': %v", outputDir, err)
		}
	}
}

// FormatResultsToText formats the SimulationResults into a human-readable string.
func FormatResultsToText(results *SimulationResults) string {
	var sb strings.Builder

	// Ensure results and its timestamp are not nil before using them
	timestampStr := "N/A"
	if results != nil && results.Timestamp != "" {
		timestampStr = results.Timestamp
	} else if results != nil {
		timestampStr = "Timestamp not set"
	}


	sb.WriteString(fmt.Sprintf("Simulation Results - Timestamp: %s\n", timestampStr))
	sb.WriteString("==================================================\n\n")

	if results == nil {
		sb.WriteString("No simulation data available.\n")
		sb.WriteString("==================================================\n")
		sb.WriteString("End of Report\n")
		return sb.String()
	}

	sb.WriteString("Input Parameters:\n")
	sb.WriteString(fmt.Sprintf("  Input Text: %s\n", results.InputText))
	sb.WriteString(fmt.Sprintf("  Gold Code N: %d\n", results.GoldN))
	sb.WriteString(fmt.Sprintf("  Gold Taps1: %v\n", results.GoldTaps1))
	sb.WriteString(fmt.Sprintf("  Gold Taps2: %v\n", results.GoldTaps2))
	sb.WriteString(fmt.Sprintf("  Decoder Type: %s\n", results.DecoderType))
	sb.WriteString(fmt.Sprintf("  Error Type: %s\n", results.ErrorType))
	sb.WriteString(fmt.Sprintf("  Error Rate: %.2f%%\n", results.ErrorRate))
	sb.WriteString("\n")

	sb.WriteString("Generated/Processed Sequences:\n")
	if results.Original != nil {
		sb.WriteString(fmt.Sprintf("  Original Sequence (length %d):\n    %s\n", results.Original.Len(), results.Original.String()))
		if results.InputText != "" {
			sb.WriteString(fmt.Sprintf("  Original ASCII: %s\n", bitsToASCII(results.Original.String())))
		}
	} else {
		sb.WriteString("  Original Sequence: Not available\n")
	}

	if results.GoldCode != nil {
		sb.WriteString(fmt.Sprintf("  Gold Code (length %d):\n    %s\n", results.GoldCode.Len(), results.GoldCode.String()))
	} else {
		sb.WriteString("  Gold Code: Not available\n")
	}

	if results.Encoded != nil {
		sb.WriteString(fmt.Sprintf("  Encoded Sequence (length %d):\n    %s\n", results.Encoded.Len(), results.Encoded.String()))
	} else {
		sb.WriteString("  Encoded Sequence: Not available / Module disabled\n")
	}

	if results.Corrupted != nil {
		sb.WriteString(fmt.Sprintf("  Corrupted Sequence (length %d):\n    %s\n", results.Corrupted.Len(), results.Corrupted.String()))
		sb.WriteString(fmt.Sprintf("  Errors Introduced: %d\n", results.ErrorsIntroduced))
	} else {
		sb.WriteString("  Corrupted Sequence: Not available / Error module disabled\n")
	}

	if results.Decoded != nil {
		sb.WriteString(fmt.Sprintf("  Decoded Sequence (length %d):\n    %s\n", results.Decoded.Len(), results.Decoded.String()))
		if results.InputText != "" { // Only show decoded ASCII if input was text
			sb.WriteString(fmt.Sprintf("  Decoded ASCII: %s\n", bitsToASCII(results.Decoded.String())))
		}
	} else {
		sb.WriteString("  Decoded Sequence: Not available / Decoder module disabled\n")
	}
	sb.WriteString("\n")

	sb.WriteString("Analysis Results:\n")
	if results.Original != nil && results.Decoded != nil { // BER requires Original and Decoded
		sb.WriteString(fmt.Sprintf("  BER: %.4f (%.2f%%)\n", results.BER, results.BER*100))
		sb.WriteString(fmt.Sprintf("  Error Count (vs Original): %d / %d bits\n", results.ErrorCount, results.Original.Len()))
	} else {
		sb.WriteString("  BER: Not calculated / Relevant modules disabled\n")
		sb.WriteString("  Error Count: Not calculated\n")
	}
	sb.WriteString("\n")

	sb.WriteString("Autocorrelation (Max Absolute Off-Peak):\n")
	if results.OriginalAutocorr != 0 || results.EncodedAutocorr != 0 || results.CorruptedAutocorr != 0 || (results.Original != nil) {
		sb.WriteString(fmt.Sprintf("  Original Sequence: %.4f\n", results.OriginalAutocorr))
		if results.Encoded != nil {
			sb.WriteString(fmt.Sprintf("  Encoded Sequence: %.4f\n", results.EncodedAutocorr))
		} else {
			sb.WriteString("  Encoded Sequence: N/A (sequence not generated)\n")
		}
		if results.Corrupted != nil {
			sb.WriteString(fmt.Sprintf("  Corrupted Sequence: %.4f\n", results.CorruptedAutocorr))
		} else {
			sb.WriteString("  Corrupted Sequence: N/A (sequence not generated)\n")
		}
	} else {
		sb.WriteString("  Autocorrelation analysis was not performed or results are zero.\n")
	}

	sb.WriteString("\n==================================================\n")
	sb.WriteString("End of Report\n")

	return sb.String()
}

// SaveResultsToFileOnServer saves the formatted results to a timestamped file.
// Returns the path to the saved file and any error encountered.
func SaveResultsToFileOnServer(results *SimulationResults) (string, error) {
	ensureOutputDirExists()

	if results == nil {
		return "", fmt.Errorf("cannot save nil results")
	}

	// Create a filename based on the simulation timestamp (results.Timestamp)
	var t time.Time
	var errParse error
	if results.Timestamp != "" {
		t, errParse = time.Parse(time.RFC1123, results.Timestamp)
		if errParse != nil {
			log.Printf("Warning: Could not parse results.Timestamp ('%s') for filename, using current time: %v", results.Timestamp, errParse)
			t = time.Now()
		}
	} else {
		log.Println("Warning: results.Timestamp is empty, using current time for filename.")
		t = time.Now() // Fallback if timestamp is empty
	}

	filename := fmt.Sprintf("simulation_results_%s.txt", t.Format("20060102_150405.000"))
	filePath := filepath.Join(outputDir, filename)

	// Format the results into a string
	content := FormatResultsToText(results) // Use the local (or package-level) formatting function

	// Write the content to the file
	err := os.WriteFile(filePath, []byte(content), 0644) // Read/Write for owner, Read for group/others
	if err != nil {
		log.Printf("Error writing results to file '%s': %v", filePath, err)
		return "", err
	}

	log.Printf("Simulation results saved to server at: %s", filePath)
	cleanupOldResultFiles()
	return filePath, nil
}

// cleanupOldResultFiles ensures that only the 'maxResultFiles' most recent result files are kept.
func cleanupOldResultFiles() {
	files, err := os.ReadDir(outputDir)
	if err != nil {
		log.Printf("Error reading output directory '%s' for cleanup: %v", outputDir, err)
		return
	}

	resultFiles := []os.DirEntry{}
	for _, file := range files {
		// Filter for our specific result files
		if !file.IsDir() && strings.HasPrefix(file.Name(), "simulation_results_") && strings.HasSuffix(file.Name(), ".txt") {
			resultFiles = append(resultFiles, file)
		}
	}

	if len(resultFiles) <= maxResultFiles {
		return // No need to clean up
	}

	// Sort files by modification time (oldest first).
	sort.Slice(resultFiles, func(i, j int) bool {
		infoI, errI := resultFiles[i].Info()
		infoJ, errJ := resultFiles[j].Info()
		if errI != nil || errJ != nil {
			return false
		}
		return infoI.ModTime().Before(infoJ.ModTime())
	})


	filesToDelete := len(resultFiles) - maxResultFiles
	for i := range filesToDelete {
		filePathToDelete := filepath.Join(outputDir, resultFiles[i].Name())
		err := os.Remove(filePathToDelete)
		if err != nil {
			log.Printf("Error deleting old result file '%s': %v", filePathToDelete, err)
		} else {
			log.Printf("Deleted old result file: %s", filePathToDelete)
		}
	}
}
