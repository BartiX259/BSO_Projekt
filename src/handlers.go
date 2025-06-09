package src

import (
	"fmt"
	"html/template"
	"log"
	"math" // Added import
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/BartiX259/BSO_Projekt/src/simulation"
)

// Global simulation results storage
type SimulationResults struct {
	Original         *simulation.BitSequence
	GoldCode         *simulation.BitSequence
	Encoded          *simulation.BitSequence
	Corrupted        *simulation.BitSequence
	Decoded          *simulation.BitSequence
	BER              float32
	ErrorCount       int
	InputText        string
	ErrorType        string
	ErrorRate        float64
	ErrorsIntroduced int
	Timestamp        string
	// Add user parameters
	GoldN       int
	GoldTaps1   []uint
	GoldTaps2   []uint
	DecoderType string
	// Add autocorrelation analysis
	OriginalAutocorr  float32
	EncodedAutocorr   float32
	CorruptedAutocorr float32
	mutex             sync.RWMutex
}

var globalResults = &SimulationResults{}

var latestGeneralSimFilePath string
var latestGeneralSimFileMutex sync.RWMutex

var latestCDMASimFilePath string
var latestCDMASimFileMutex sync.RWMutex
// --- NEW: CDMA Simulation global storage ---
// var cdmaGlobalResults = struct { // This will be replaced
// 	sync.RWMutex
// 	Result *simulation.CDMAResult
// }{}

type CDMASimulationState struct {
	mutex sync.RWMutex

	// Module 1: System Config (Inputs are from form, results are derived properties)
	GlobalN     uint
	GlobalPoly1 []uint
	GlobalPoly2 []uint
	Timestamp   string // General timestamp for the whole simulation set

	// Module 2: Transmitters
	// User A
	InputTextA         string
	SeedA1_form        uint64 // Seed for LFSR1 for User A's Gold Code from form
	SeedA2_form        uint64 // Seed for LFSR2 for User A's Gold Code from form
	OriginalDataStrA   string // Bit string of User A's data
	EncodedDataStrA    string // Bit string of User A's encoded data
	DataLengthA        int    // Length of User A's data in bits
	GeneratedGoldCodeA string // User A's Gold Code string
	// User B
	InputTextB         string
	SeedB1_form        uint64 // Seed for LFSR1 for User B's Gold Code from form
	SeedB2_form        uint64 // Seed for LFSR2 for User B's Gold Code from form
	OriginalDataStrB   string // Bit string of User B's data
	EncodedDataStrB    string // Bit string of User B's encoded data
	DataLengthB        int    // Length of User B's data in bits
	GeneratedGoldCodeB string // User B's Gold Code string

	SimulationDataLength int // Max of DataLengthA, DataLengthB

	// Module 3: Channel
	NoiseLevel_form           float64 // Noise level from form
	TransmittedSignalAStr     string  // NEW: User A transmitted signal
	TransmittedSignalBStr     string  // NEW: User B transmitted signal
	CombinedSignalStr         string
	ReceivedSignalStr         string
	GoldCodeLength            int    // Needed for context in channel display
	ReceivedSignalSegmentAStr string // NEW
	ReceivedSignalSegmentBStr string // NEW
	CorrelatedSignalAStr      string // NEW
	CorrelatedSignalBStr      string // NEW

	// Module 4: Receivers
	// User A
	DecodedTextA    string
	DecodedDataStrA string
	ErrorCountA     int
	BER_A_str       string
	// User B
	DecodedTextB    string
	DecodedDataStrB string
	ErrorCountB     int
	BER_B_str       string

	// Module 5: Code Analysis
	AutocorrelationPeak        int     // GoldCodeLength
	MaxOffPeakAutocorrelationA float32 // Normalized
	MaxOffPeakAutocorrelationB float32 // Normalized
	CrossCorrelationAB         float32 // Normalized
}

var cdmaGlobalState = &CDMASimulationState{}

// --- END NEW ---

// ResponseData holds data for the response template
type ResponseData struct {
	Timestamp       string
	BitSequence     string
	EncodedSequence string
	DecodedSequence string
	BER             string
}

// GeneratorData holds data for generator template
type GeneratorData struct {
	BitSequence string
	InputText   string
	Length      int
}

// EncoderData holds data for encoder template
type EncoderData struct {
	GoldCode        string
	EncodedSequence string
	N               int
	Length          int
	Taps1           []uint
	Taps2           []uint
}

// ErrorData holds data for error template
type ErrorData struct {
	CorruptedSequence string
	ErrorType         string
	ErrorRate         float64
	ErrorsIntroduced  int
}

// DecoderData holds data for decoder template
type DecoderData struct {
	DecodedSequence string
	DecoderType     string
	DecodedASCII    string
}

// BERData holds data for BER template
type BERData struct {
	BER              string
	ErrorsDetected   int
	TotalBits        int
	OriginalSequence string
	DecodedSequence  string
	OriginalASCII    string
	DecodedASCII     string
}

// AutocorrelationData holds data for autocorrelation template
type AutocorrelationData struct {
	OriginalMaxOffPeak  string
	EncodedMaxOffPeak   string
	CorruptedMaxOffPeak string
}

// --- NEW: CDMA Handler Data Structs ---
type CDMAFormData struct { // Matches form fields
	GoldNStr     string // Mod 1
	GoldTaps1Str string // Mod 1
	GoldTaps2Str string // Mod 1

	TextUserAStr string // Mod 2A
	SeedA1Str    string // Mod 2A
	SeedA2Str    string // Mod 2A

	TextUserBStr string // Mod 2B
	SeedB1Str    string // Mod 2B
	SeedB2Str    string // Mod 2B

	SeqLengthRandomStr string // Fallback if texts are empty (common for A & B if both random)

	NoiseLevelStr string // Mod 3
}

// Data structs for individual CDMA result templates (Module specific)
type CDMASystemConfigData struct { // For Module 1 results display
	Timestamp                  string
	GlobalN                    uint
	GlobalPoly1                []uint
	GlobalPoly2                []uint
	GeneratedGoldCodeA         string // Display part of Gold Code A
	GeneratedGoldCodeB         string // Display part of Gold Code B
	GoldCodeLength             int
	MaxOffPeakAutocorrelationA float32
	MaxOffPeakAutocorrelationB float32
	CrossCorrelationAB         float32
}

type CDMATransmitterUserData struct { // For Module 2 results (User A or B)
	Timestamp       string
	UserLabel       string // "A" or "B"
	InputText       string
	Seed1           uint64
	Seed2           uint64
	OriginalDataStr string
	EncodedDataStr  string // NEW: Encoded data string
	DataLength      int
	GoldCodeLength  int // For context
}

type CDMAChannelData struct { // For Module 3 results
	Timestamp         string
	NoiseLevel        float64
	CombinedSignalStr string
	ReceivedSignalStr string
	DataBitLength     int // SimulationDataLength
	GoldCodeLength    int
}

type CDMAReceiverUserData struct { // For Module 4 results (User A or B)
	Timestamp                string
	UserLabel                string // "A" or "B"
	InputText                string // Original text
	DecodedText              string
	OriginalDataStr          string // For comparison if needed
	DecodedDataStr           string
	ErrorCount               int
	BER_str                  string
	DataLength               int
	ReceivedSignalSegmentStr string // NEW: Received signal segment for this user
	CorrelatedSignalStr      string // NEW: Correlated signal for this user
}

type CDMACodeAnalysisData struct { // For Module 5 results
	Timestamp                  string
	AutocorrelationPeak        int
	MaxOffPeakAutocorrelationA float32
	MaxOffPeakAutocorrelationB float32
	CrossCorrelationAB         float32
	GoldCodeLength             int // For context (same as AutocorrelationPeak)
}

// --- END NEW ---

// Serve the main HTML page using a template (Exported)
func IndexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/index.html") // Path relative to execution dir
	if err != nil {
		log.Printf("Error parsing index template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(w, nil)
	if err != nil {
		log.Printf("Error executing index template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// DownloadGeneralSimResultsHandler serves the latest general simulation results file.
func DownloadGeneralSimResultsHandler(w http.ResponseWriter, r *http.Request) {
	latestGeneralSimFileMutex.RLock()
	currentFilePath := latestGeneralSimFilePath
	latestGeneralSimFileMutex.RUnlock()

	if currentFilePath == "" {
		http.Error(w, "No general simulation results saved yet. Run a general simulation first.", http.StatusNotFound)
		return
	}

	serveFileForDownload(w, r, currentFilePath)
}

// DownloadCDMASimResultsHandler serves the latest CDMA simulation results file.
func DownloadCDMASimResultsHandler(w http.ResponseWriter, r *http.Request) {
	latestCDMASimFileMutex.RLock()
	currentFilePath := latestCDMASimFilePath
	latestCDMASimFileMutex.RUnlock()

	if currentFilePath == "" {
		http.Error(w, "No CDMA simulation results saved yet. Run a CDMA simulation first.", http.StatusNotFound)
		return
	}

	serveFileForDownload(w, r, currentFilePath)
}

// serveFileForDownload is a helper to reduce duplication
func serveFileForDownload(w http.ResponseWriter, r *http.Request, filePath string) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("serveFileForDownload: File '%s' not found on server.", filePath)
		http.Error(w, "Saved results file not found. It may have been deleted. Please run a new simulation.", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename=\""+filepath.Base(filePath)+"\"")
	http.ServeFile(w, r, filePath)
}

// Handle the simulation endpoint - runs complete simulation pipeline and stores results globally
func SimulateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("Form parse error: %v", err)
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	// Debug logging - log all received form values
	log.Printf("=== Form Values Received ===")
	log.Printf("Request Method: %s", r.Method)
	log.Printf("Content-Type: %s", r.Header.Get("Content-Type"))

	// Parse form data with better error handling
	seqType := strings.TrimSpace(r.FormValue("seqType"))
	seqText := strings.TrimSpace(r.FormValue("seqText"))
	seqLengthStr := strings.TrimSpace(r.FormValue("seqLength"))
	errorRateStr := strings.TrimSpace(r.FormValue("errorRate"))
	errorType := strings.TrimSpace(r.FormValue("errorType"))
	goldNStr := strings.TrimSpace(r.FormValue("goldN"))
	goldTaps1Str := strings.TrimSpace(r.FormValue("goldTaps1"))
	goldTaps2Str := strings.TrimSpace(r.FormValue("goldTaps2"))
	decoderType := strings.TrimSpace(r.FormValue("decoderType"))

	// Parse module enable checkboxes
	errorEnabled := r.FormValue("errorEnabled") == "on"
	decoderEnabled := r.FormValue("decoderEnabled") == "on"
	berEnabled := r.FormValue("berEnabled") == "on"
	autocorrEnabled := r.FormValue("autocorrEnabled") == "on"

	log.Printf("seqText: '%s'", seqText)
	log.Printf("seqLength: '%s'", seqLengthStr)
	log.Printf("errorRate: '%s'", errorRateStr)
	log.Printf("errorType: '%s'", errorType)
	log.Printf("goldN: '%s'", goldNStr)
	log.Printf("goldTaps1: '%s'", goldTaps1Str)
	log.Printf("goldTaps2: '%s'", goldTaps2Str)
	log.Printf("decoderType: '%s'", decoderType)
	log.Printf("=== End Form Values ===")

	// Generate or parse bit sequence
	var bitSeq *simulation.BitSequence
	if seqType == "text" {
		bitSeq = simulation.StringAsSequence(seqText)
		log.Printf("Using text input: '%s'", seqText)
	} else {
		seqLength := 64
		if seqLengthStr != "" {
			if parsed, err := strconv.Atoi(seqLengthStr); err == nil && parsed > 0 {
				seqLength = parsed
			}
		}
		if seqType == "random-text" {
			seqType = "text"
			seqText = simulation.RandomText(seqLength)
			bitSeq = simulation.StringAsSequence(seqText)
		} else {
			bitSeq = simulation.RandomSequence(seqLength)
		}
		log.Printf("Generated random sequence of length: %d", seqLength)
	}

	// Parse Gold code parameters with defaults
	n := 10
	if goldNStr != "" {
		if parsed, err := strconv.Atoi(goldNStr); err == nil && parsed >= 2 && parsed <= 16 {
			n = parsed
		}
	}
	log.Printf("Using Gold N: %d", n)

	// Parse taps for LFSR1
	taps1 := []uint{0, 3} // default
	if goldTaps1Str != "" {
		if parsed := parseTaps(goldTaps1Str); len(parsed) > 0 {
			taps1 = parsed
		}
	}
	log.Printf("Using LFSR1 taps: %v", taps1)

	// Parse taps for LFSR2
	taps2 := []uint{0, 2, 3, 8} // default
	if goldTaps2Str != "" {
		if parsed := parseTaps(goldTaps2Str); len(parsed) > 0 {
			taps2 = parsed
		}
	}
	log.Printf("Using LFSR2 taps: %v", taps2)

	// Parse error parameters with better validation
	errorRate := 5.0
	if errorRateStr != "" {
		if parsed, err := strconv.ParseFloat(errorRateStr, 64); err == nil && parsed >= 0 && parsed <= 100 {
			errorRate = parsed
		} else {
			log.Printf("Error parsing errorRate '%s': %v, using default 5.0", errorRateStr, err)
		}
	}
	log.Printf("Using error rate: %.2f%%", errorRate)

	if errorType == "" {
		errorType = "random"
	}
	log.Printf("Using error type: %s", errorType)

	if decoderType == "" {
		decoderType = "xor"
	}
	log.Printf("Using decoder type: %s", decoderType)

	// Generate Gold code with user parameters
	seed1 := uint64(1)
	seed2 := uint64(0b1010101010)
	goldCode := simulation.GenerateGoldCode(uint(n), taps1, seed1, taps2, seed2)

	// Conditional simulation pipeline
	var encoded *simulation.BitSequence
	if goldCode != nil {
		encodedTmp := simulation.EncodeWithGold(*bitSeq, *goldCode)
		encoded = encodedTmp
	} else {
		encoded = nil
	}

	var corrupted *simulation.BitSequence
	var errorsIntroduced int
	if errorEnabled && encoded != nil {
		errorRateDecimal := errorRate / 100.0
		corruptedTmp, errors := simulation.AddErrors(encoded, errorRateDecimal, errorType)
		corrupted = corruptedTmp
		errorsIntroduced = errors
	} else if encoded != nil {
		// If error module is disabled, pass encoded as corrupted (no errors)
		corrupted = encoded
		errorsIntroduced = 0
	} else {
		corrupted = nil
		errorsIntroduced = 0
	}

	var decoded *simulation.BitSequence
	if decoderEnabled && corrupted != nil && goldCode != nil {
		decodedTmp := simulation.DecodeWithGold(*corrupted, *goldCode)
		decoded = decodedTmp
	} else {
		decoded = nil
	}

	var ber float32
	var errorCount int
	if berEnabled && decoded != nil {
		ber = simulation.CalculateBER(*bitSeq, *decoded)
		// Count errors for display
		errorCount = 0
		for i := range bitSeq.Len() {
			if bitSeq.Get(i) != decoded.Get(i) {
				errorCount++
			}
		}
	} else {
		ber = 0
		errorCount = 0
	}

	var originalAutocorr, encodedAutocorr, corruptedAutocorr float32
	if autocorrEnabled {
		originalAutocorr = simulation.MaxAbsoluteOffPeak(simulation.CalculatePeriodicAutocorrelation(*bitSeq))
		if encoded != nil {
			encodedAutocorr = simulation.MaxAbsoluteOffPeak(simulation.CalculatePeriodicAutocorrelation(*encoded))
		}
		if corrupted != nil {
			corruptedAutocorr = simulation.MaxAbsoluteOffPeak(simulation.CalculatePeriodicAutocorrelation(*corrupted))
		}
	}

	// Store results globally for other handlers to access
	globalResults.mutex.Lock()
	globalResults.Original = bitSeq
	globalResults.GoldCode = goldCode
	globalResults.Encoded = encoded
	globalResults.Corrupted = corrupted
	globalResults.Decoded = decoded
	globalResults.BER = ber
	globalResults.ErrorCount = errorCount
	if seqType == "text" {
		globalResults.InputText = seqText
	} else {
		globalResults.InputText = ""
	}
	globalResults.ErrorType = errorType
	globalResults.ErrorRate = errorRate
	globalResults.ErrorsIntroduced = errorsIntroduced
	globalResults.Timestamp = time.Now().Format(time.RFC1123)
	// Store user parameters
	globalResults.GoldN = n
	globalResults.GoldTaps1 = taps1
	globalResults.GoldTaps2 = taps2
	globalResults.DecoderType = decoderType
	// Store autocorrelation results
	globalResults.OriginalAutocorr = originalAutocorr
	globalResults.EncodedAutocorr = encodedAutocorr
	globalResults.CorruptedAutocorr = corruptedAutocorr
	globalResults.mutex.Unlock()

	savedPath, err := SaveSimulationResultsToFile(globalResults)
	if err != nil {
		log.Printf("Error saving simulation results to file: %v", err)
	} else {
		latestGeneralSimFileMutex.Lock()
		latestGeneralSimFilePath = savedPath
		latestGeneralSimFileMutex.Unlock()
	}

	// Return success response with HTMX trigger event
	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Trigger", "simulation-complete")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<div class="success-message">Symulacja zakończona pomyślnie! Czas: %s</div>`,
		time.Now().Format("15:04:05"))
}

// GeneratorHandler returns stored bit sequence generation results
func GeneratorHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("GeneratorHandler called")

	// Get stored results from global state
	globalResults.mutex.RLock()

	// Check if we have stored results
	if globalResults.Original == nil {
		globalResults.mutex.RUnlock()
		log.Printf("GeneratorHandler: No original sequence available")
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}

	data := GeneratorData{
		BitSequence: globalResults.Original.String(),
		InputText:   globalResults.InputText,
		Length:      globalResults.Original.Len(),
	}
	globalResults.mutex.RUnlock()

	log.Printf("GeneratorHandler: Returning data - Length: %d, InputText: '%s', BitSequence preview: '%.20s...'",
		data.Length, data.InputText, data.BitSequence)

	tmpl, err := template.ParseFiles("templates/generator_result.html")
	if err != nil {
		log.Printf("GeneratorHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-cache")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing generator template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	} else {
		log.Printf("GeneratorHandler: Template executed successfully")
	}
}

// EncoderHandler returns stored Gold code generation and encoding results
func EncoderHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("EncoderHandler called")

	// Get stored results from global state
	globalResults.mutex.RLock()

	// Check if we have stored results
	if globalResults.GoldCode == nil || globalResults.Encoded == nil {
		globalResults.mutex.RUnlock()
		log.Printf("EncoderHandler: No gold code or encoded sequence available")
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}

	data := EncoderData{
		GoldCode:        globalResults.GoldCode.String(),
		EncodedSequence: globalResults.Encoded.String(),
		N:               globalResults.GoldN, // Use actual user parameter
		Length:          globalResults.GoldCode.Len(),
		Taps1:           globalResults.GoldTaps1,
		Taps2:           globalResults.GoldTaps2,
	}
	globalResults.mutex.RUnlock()

	log.Printf("EncoderHandler: Returning data - N: %d, Length: %d, Taps1: %v, Taps2: %v",
		data.N, data.Length, data.Taps1, data.Taps2)

	tmpl, err := template.ParseFiles("templates/encoder_result.html")
	if err != nil {
		log.Printf("EncoderHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-cache")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing encoder template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	} else {
		log.Printf("EncoderHandler: Template executed successfully")
	}
}

// ErrorHandler returns stored error injection results
func ErrorHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("ErrorHandler called")

	globalResults.mutex.RLock()
	if globalResults.Encoded != nil && globalResults.Corrupted == globalResults.Encoded {
		// Moduł wyłączony
		globalResults.mutex.RUnlock()
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="module-disabled-message">Moduł dodawania błędów jest wyłączony.</div>`)
		return
	}
	if globalResults.Corrupted == nil {
		globalResults.mutex.RUnlock()
		log.Printf("ErrorHandler: No corrupted sequence available")
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}

	data := ErrorData{
		CorruptedSequence: globalResults.Corrupted.String(),
		ErrorType:         globalResults.ErrorType,
		ErrorRate:         globalResults.ErrorRate,
		ErrorsIntroduced:  globalResults.ErrorsIntroduced,
	}
	globalResults.mutex.RUnlock()

	log.Printf("ErrorHandler: Returning data - ErrorType: %s, ErrorRate: %.2f, ErrorsIntroduced: %d",
		data.ErrorType, data.ErrorRate, data.ErrorsIntroduced)

	tmpl, err := template.ParseFiles("templates/error_result.html")
	if err != nil {
		log.Printf("ErrorHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing error template: %v", err)
	}
}

// DecoderHandler returns stored decoding results
func DecoderHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("DecoderHandler called")

	globalResults.mutex.RLock()
	if globalResults.Decoded == nil {
		globalResults.mutex.RUnlock()
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="module-disabled-message">Moduł dekodera jest wyłączony.</div>`)
		return
	}

	decodedBits := globalResults.Decoded.String()
	ascii := ""
	// Only show ASCII if input was text (not random)
	if globalResults.InputText != "" {
		ascii = bitsToASCII(decodedBits)
	}

	data := DecoderData{
		DecodedSequence: decodedBits,
		DecoderType:     globalResults.DecoderType,
		DecodedASCII:    ascii,
	}
	globalResults.mutex.RUnlock()

	log.Printf("DecoderHandler: Returning data - DecoderType: %s", data.DecoderType)

	tmpl, err := template.ParseFiles("templates/decoder_result.html")
	if err != nil {
		log.Printf("DecoderHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing decoder template: %v", err)
	}
}

// BERHandler returns stored BER calculation results
func BERHandler(w http.ResponseWriter, r *http.Request) {
	globalResults.mutex.RLock()
	if globalResults.Original == nil {
		globalResults.mutex.RUnlock()
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}
	if globalResults.Decoded == nil {
		globalResults.mutex.RUnlock()
		w.Header().Set("Content-Type", "text/html")
		// Sprawdź czy dekoder był wyłączony
		msg := `<div class="module-disabled-message">Moduł BER jest wyłączony.</div>`
		if globalResults.Decoded == nil {
			msg = `<div class="module-disabled-message">Moduł BER wymaga działania modułu dekodera.</div>`
		}
		fmt.Fprint(w, msg)
		return
	}

	origBits := globalResults.Original.String()
	decBits := globalResults.Decoded.String()

	origASCII := ""
	decASCII := ""
	// Only show ASCII if input was text (not random)
	if globalResults.InputText != "" {
		origASCII = bitsToASCII(origBits)
		decASCII = bitsToASCII(decBits)
	}

	data := BERData{
		BER:              fmt.Sprintf("%.2f", globalResults.BER*100),
		ErrorsDetected:   globalResults.ErrorCount,
		TotalBits:        globalResults.Original.Len(),
		OriginalSequence: origBits,
		DecodedSequence:  decBits,
		OriginalASCII:    origASCII,
		DecodedASCII:     decASCII,
	}
	globalResults.mutex.RUnlock()

	tmpl, err := template.ParseFiles("templates/ber_result.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing BER template: %v", err)
	}
}

// AutocorrelationHandler returns stored autocorrelation analysis results
func AutocorrelationHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("AutocorrelationHandler called")

	// Get stored results from global state
	globalResults.mutex.RLock()

	// Check if we have stored results
	if globalResults.Original == nil {
		globalResults.mutex.RUnlock()
		log.Printf("AutocorrelationHandler: No simulation results available")
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}
	// Sprawdź czy autokorelacja była liczona (wszystkie wartości == 0)
	if globalResults.OriginalAutocorr == 0 && globalResults.EncodedAutocorr == 0 && globalResults.CorruptedAutocorr == 0 {
		globalResults.mutex.RUnlock()
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<div class="module-disabled-message">Moduł autokorelacji jest wyłączony.</div>`)
		return
	}

	data := AutocorrelationData{
		OriginalMaxOffPeak:  fmt.Sprintf("%.4f", globalResults.OriginalAutocorr),
		EncodedMaxOffPeak:   fmt.Sprintf("%.4f", globalResults.EncodedAutocorr),
		CorruptedMaxOffPeak: fmt.Sprintf("%.4f", globalResults.CorruptedAutocorr),
	}
	globalResults.mutex.RUnlock()

	log.Printf("AutocorrelationHandler: Returning data - Original: %.4f, Encoded: %.4f, Corrupted: %.4f",
		globalResults.OriginalAutocorr, globalResults.EncodedAutocorr, globalResults.CorruptedAutocorr)

	tmpl, err := template.ParseFiles("templates/autocorrelation_result.html")
	if err != nil {
		log.Printf("AutocorrelationHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("Cache-Control", "no-cache")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing autocorrelation template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	} else {
		log.Printf("AutocorrelationHandler: Template executed successfully")
	}
}

// --- NEW: CDMA Simulation Handlers ---

// CDMASimulateHandler runs the CDMA simulation and stores results.
func CDMASimulateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		log.Printf("CDMA Form parse error: %v", err)
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	formData := CDMAFormData{
		GoldNStr:           r.FormValue("cdmaGoldN"),
		GoldTaps1Str:       r.FormValue("cdmaGoldTaps1"),
		GoldTaps2Str:       r.FormValue("cdmaGoldTaps2"),
		TextUserAStr:       strings.TrimSpace(r.FormValue("cdmaTextUserA")),
		SeedA1Str:          r.FormValue("cdmaSeedA1"),
		SeedA2Str:          r.FormValue("cdmaSeedA2"),
		TextUserBStr:       strings.TrimSpace(r.FormValue("cdmaTextUserB")),
		SeedB1Str:          r.FormValue("cdmaSeedB1"),
		SeedB2Str:          r.FormValue("cdmaSeedB2"),
		SeqLengthRandomStr: r.FormValue("cdmaSeqLengthRandom"),
		NoiseLevelStr:      r.FormValue("cdmaNoiseLevel"),
	}
	log.Printf("CDMA Simulation Form Data: %+v", formData)

	// Parse Module 1 inputs
	goldN := uint(parseIntWithDefault(formData.GoldNStr, 10, 2, 16))
	taps1 := parseTapsWithDefault(formData.GoldTaps1Str, []uint{0, 3})
	taps2 := parseTapsWithDefault(formData.GoldTaps2Str, []uint{0, 2, 3, 8})

	// Parse Module 2 inputs
	seedA1 := parseUint64WithDefault(formData.SeedA1Str, 1)
	seedA2 := parseUint64WithDefault(formData.SeedA2Str, 1)
	seedB1 := parseUint64WithDefault(formData.SeedB1Str, 2)
	seedB2 := parseUint64WithDefault(formData.SeedB2Str, 2)

	seqLengthRandomBytes := parseIntWithDefault(formData.SeqLengthRandomStr, 1, 1, 10)
	seqLengthRandomBits := seqLengthRandomBytes * 8

	// Parse Module 3 input - convert percentage to decimal
	// Default to 100%, min 0%, max 10000% (allowing std dev up to 100.0)
	noiseLevelPercent := parseFloatWithDefault(formData.NoiseLevelStr, 100.0, 0.0, math.MaxFloat64) // Changed maxVal to math.MaxFloat64
	noiseLevel := noiseLevelPercent / 100.0                                                         // Convert percentage to decimal std dev

	// Perform CDMA simulation
	simResult := simulation.SimulateCDMA(
		goldN, taps1, taps2,
		seedA1, seedA2, formData.TextUserAStr,
		seedB1, seedB2, formData.TextUserBStr,
		seqLengthRandomBits,
		noiseLevel,
	)

	// Store the percentage value for display
	simResult.NoiseLevel = noiseLevelPercent

	// Populate cdmaGlobalState (same as before)
	cdmaGlobalState.mutex.Lock()
	cdmaGlobalState.Timestamp = simResult.Timestamp
	cdmaGlobalState.GlobalN = simResult.N
	cdmaGlobalState.GlobalPoly1 = simResult.Poly1
	cdmaGlobalState.GlobalPoly2 = simResult.Poly2
	cdmaGlobalState.InputTextA = simResult.InputTextA
	cdmaGlobalState.SeedA1_form = simResult.SeedA1
	cdmaGlobalState.SeedA2_form = simResult.SeedA2
	if simResult.OriginalDataSeqA != nil {
		cdmaGlobalState.OriginalDataStrA = simResult.OriginalDataSeqA.String()
	}
	if simResult.EncodedDataSeqA != nil {
		cdmaGlobalState.EncodedDataStrA = simResult.EncodedDataSeqA.String()
	}
	cdmaGlobalState.DataLengthA = simResult.DataBitLengthUserA
	cdmaGlobalState.GeneratedGoldCodeA = simResult.GoldCodeAStr

	cdmaGlobalState.InputTextB = simResult.InputTextB
	cdmaGlobalState.SeedB1_form = simResult.SeedB1
	cdmaGlobalState.SeedB2_form = simResult.SeedB2
	if simResult.OriginalDataSeqB != nil {
		cdmaGlobalState.OriginalDataStrB = simResult.OriginalDataSeqB.String()
	}
	if simResult.EncodedDataSeqB != nil {
		cdmaGlobalState.EncodedDataStrB = simResult.EncodedDataSeqB.String()
	}
	cdmaGlobalState.DataLengthB = simResult.DataBitLengthUserB
	cdmaGlobalState.GeneratedGoldCodeB = simResult.GoldCodeBStr

	cdmaGlobalState.SimulationDataLength = simResult.SimulationDataLength
	cdmaGlobalState.NoiseLevel_form = simResult.NoiseLevel
	cdmaGlobalState.TransmittedSignalAStr = simResult.TransmittedSignalAStr
	cdmaGlobalState.TransmittedSignalBStr = simResult.TransmittedSignalBStr
	cdmaGlobalState.CombinedSignalStr = simResult.CombinedSignalStr
	cdmaGlobalState.ReceivedSignalStr = simResult.ReceivedSignalStr
	cdmaGlobalState.GoldCodeLength = simResult.GoldCodeLength
	cdmaGlobalState.ReceivedSignalSegmentAStr = simResult.ReceivedSignalSegmentAStr
	cdmaGlobalState.ReceivedSignalSegmentBStr = simResult.ReceivedSignalSegmentBStr
	cdmaGlobalState.CorrelatedSignalAStr = simResult.CorrelatedSignalUserAStr // NEW
	cdmaGlobalState.CorrelatedSignalBStr = simResult.CorrelatedSignalUserBStr // NEW

	cdmaGlobalState.DecodedTextA = simResult.DecodedTextA
	if simResult.DecodedDataSeqA != nil {
		cdmaGlobalState.DecodedDataStrA = simResult.DecodedDataSeqA.String()
	}
	cdmaGlobalState.ErrorCountA = simResult.ErrorCountA
	cdmaGlobalState.BER_A_str = fmt.Sprintf("%.2f%%", simResult.BER_A*100)

	cdmaGlobalState.DecodedTextB = simResult.DecodedTextB
	if simResult.DecodedDataSeqB != nil {
		cdmaGlobalState.DecodedDataStrB = simResult.DecodedDataSeqB.String()
	}
	cdmaGlobalState.ErrorCountB = simResult.ErrorCountB
	cdmaGlobalState.BER_B_str = fmt.Sprintf("%.2f%%", simResult.BER_B*100)

	cdmaGlobalState.AutocorrelationPeak = simResult.AutocorrelationPeak
	cdmaGlobalState.MaxOffPeakAutocorrelationA = simResult.MaxOffPeakAutocorrelationA
	cdmaGlobalState.MaxOffPeakAutocorrelationB = simResult.MaxOffPeakAutocorrelationB
	cdmaGlobalState.CrossCorrelationAB = simResult.CrossCorrelationAB
	cdmaGlobalState.mutex.Unlock()

	savedPath, err := SaveCDMAResultsToFile(simResult)
	if err == nil {
		latestCDMASimFileMutex.Lock()
		latestCDMASimFilePath = savedPath
		latestCDMASimFileMutex.Unlock()
	}

	log.Printf("CDMA Simulation completed. Global state updated. SimDataLen: %d", simResult.SimulationDataLength)

	w.Header().Set("Content-Type", "text/html")
	w.Header().Set("HX-Trigger", "cdma-simulation-complete")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `<div class="success-message">Symulacja CDMA zakończona pomyślnie! Czas: %s</div>`, time.Now().Format("15:04:05"))
}

// Add new BER handlers for individual users
func CDMABERAResultsHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}
	data := struct {
		Timestamp   string
		UserLabel   string
		BER_str     string
		ErrorCount  int
		TotalBits   int
		InputText   string
		DecodedText string
	}{
		Timestamp:   cdmaGlobalState.Timestamp,
		UserLabel:   "A",
		BER_str:     cdmaGlobalState.BER_A_str,
		ErrorCount:  cdmaGlobalState.ErrorCountA,
		TotalBits:   cdmaGlobalState.DataLengthA,
		InputText:   cdmaGlobalState.InputTextA,
		DecodedText: cdmaGlobalState.DecodedTextA,
	}

	tmpl, err := template.ParseFiles("templates/cdma_ber_user_result.html")
	if err != nil {
		log.Printf("CDMABERAResultsHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMABERAResultsHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

func CDMABERBResultsHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}
	data := struct {
		Timestamp   string
		UserLabel   string
		BER_str     string
		ErrorCount  int
		TotalBits   int
		InputText   string
		DecodedText string
	}{
		Timestamp:   cdmaGlobalState.Timestamp,
		UserLabel:   "B",
		BER_str:     cdmaGlobalState.BER_B_str,
		ErrorCount:  cdmaGlobalState.ErrorCountB,
		TotalBits:   cdmaGlobalState.DataLengthB,
		InputText:   cdmaGlobalState.InputTextB,
		DecodedText: cdmaGlobalState.DecodedTextB,
	}

	tmpl, err := template.ParseFiles("templates/cdma_ber_user_result.html")
	if err != nil {
		log.Printf("CDMABERBResultsHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMABERBResultsHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// CDMASystemConfigHandler returns system configuration results for CDMA
func CDMASystemConfigHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}

	data := CDMASystemConfigData{
		Timestamp:                  cdmaGlobalState.Timestamp,
		GlobalN:                    cdmaGlobalState.GlobalN,
		GlobalPoly1:                cdmaGlobalState.GlobalPoly1,
		GlobalPoly2:                cdmaGlobalState.GlobalPoly2,
		GeneratedGoldCodeA:         truncateString(cdmaGlobalState.GeneratedGoldCodeA, 64),
		GeneratedGoldCodeB:         truncateString(cdmaGlobalState.GeneratedGoldCodeB, 64),
		GoldCodeLength:             cdmaGlobalState.GoldCodeLength,
		MaxOffPeakAutocorrelationA: cdmaGlobalState.MaxOffPeakAutocorrelationA,
		MaxOffPeakAutocorrelationB: cdmaGlobalState.MaxOffPeakAutocorrelationB,
		CrossCorrelationAB:         cdmaGlobalState.CrossCorrelationAB,
	}

	tmpl, err := template.ParseFiles("templates/cdma_system_config_result.html")
	if err != nil {
		log.Printf("CDMASystemConfigHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMASystemConfigHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// CDMAChannelResultsHandler returns channel results for CDMA
func CDMAChannelResultsHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}

	data := CDMAChannelData{
		Timestamp:         cdmaGlobalState.Timestamp,
		NoiseLevel:        cdmaGlobalState.NoiseLevel_form,
		CombinedSignalStr: cdmaGlobalState.CombinedSignalStr,
		ReceivedSignalStr: cdmaGlobalState.ReceivedSignalStr,
		DataBitLength:     cdmaGlobalState.SimulationDataLength,
		GoldCodeLength:    cdmaGlobalState.GoldCodeLength,
	}

	tmpl, err := template.ParseFiles("templates/cdma_channel_result.html")
	if err != nil {
		log.Printf("CDMAChannelResultsHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMAChannelResultsHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// CDMAReceiverAResultsHandler returns receiver results for User A
func CDMAReceiverAResultsHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}

	data := CDMAReceiverUserData{
		Timestamp:                cdmaGlobalState.Timestamp,
		UserLabel:                "A",
		InputText:                cdmaGlobalState.InputTextA,
		DecodedText:              cdmaGlobalState.DecodedTextA,
		OriginalDataStr:          cdmaGlobalState.OriginalDataStrA,
		DecodedDataStr:           cdmaGlobalState.DecodedDataStrA,
		ErrorCount:               cdmaGlobalState.ErrorCountA,
		BER_str:                  cdmaGlobalState.BER_A_str,
		DataLength:               cdmaGlobalState.DataLengthA,
		ReceivedSignalSegmentStr: cdmaGlobalState.ReceivedSignalSegmentAStr,
		CorrelatedSignalStr:      cdmaGlobalState.CorrelatedSignalAStr, // NEW
	}

	tmpl, err := template.ParseFiles("templates/cdma_receiver_user_result.html")
	if err != nil {
		log.Printf("CDMAReceiverAResultsHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMAReceiverAResultsHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// CDMAReceiverBResultsHandler returns receiver results for User B
func CDMAReceiverBResultsHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}

	data := CDMAReceiverUserData{
		Timestamp:                cdmaGlobalState.Timestamp,
		UserLabel:                "B",
		InputText:                cdmaGlobalState.InputTextB,
		DecodedText:              cdmaGlobalState.DecodedTextB,
		OriginalDataStr:          cdmaGlobalState.OriginalDataStrB,
		DecodedDataStr:           cdmaGlobalState.DecodedDataStrB,
		ErrorCount:               cdmaGlobalState.ErrorCountB,
		BER_str:                  cdmaGlobalState.BER_B_str,
		DataLength:               cdmaGlobalState.DataLengthB,
		ReceivedSignalSegmentStr: cdmaGlobalState.ReceivedSignalSegmentBStr,
		CorrelatedSignalStr:      cdmaGlobalState.CorrelatedSignalBStr, // NEW
	}

	tmpl, err := template.ParseFiles("templates/cdma_receiver_user_result.html")
	if err != nil {
		log.Printf("CDMAReceiverBResultsHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMAReceiverBResultsHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// CDMATransmitterAResultsHandler returns results for CDMA Transmitter User A
func CDMATransmitterAResultsHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}
	data := struct {
		Timestamp            string
		UserLabel            string
		InputText            string
		Seed1                uint64
		Seed2                uint64
		OriginalDataStr      string
		EncodedDataStr       string
		DataLength           int
		TransmittedSignalStr string // Added field
	}{
		Timestamp:            cdmaGlobalState.Timestamp,
		UserLabel:            "A",
		InputText:            cdmaGlobalState.InputTextA,
		Seed1:                cdmaGlobalState.SeedA1_form,
		Seed2:                cdmaGlobalState.SeedA2_form,
		OriginalDataStr:      cdmaGlobalState.OriginalDataStrA,
		EncodedDataStr:       cdmaGlobalState.EncodedDataStrA,
		DataLength:           cdmaGlobalState.DataLengthA,
		TransmittedSignalStr: cdmaGlobalState.TransmittedSignalAStr, // Populate added field
	}
	tmpl, err := template.ParseFiles("templates/cdma_transmitter_user_result.html")
	if err != nil {
		log.Printf("CDMATransmitterAResultsHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMATransmitterAResultsHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// CDMATransmitterBResultsHandler returns results for CDMA Transmitter User B
func CDMATransmitterBResultsHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}
	data := struct {
		Timestamp            string
		UserLabel            string
		InputText            string
		Seed1                uint64
		Seed2                uint64
		OriginalDataStr      string
		EncodedDataStr       string
		DataLength           int
		TransmittedSignalStr string // Added field
	}{
		Timestamp:            cdmaGlobalState.Timestamp,
		UserLabel:            "B",
		InputText:            cdmaGlobalState.InputTextB,
		Seed1:                cdmaGlobalState.SeedB1_form,
		Seed2:                cdmaGlobalState.SeedB2_form,
		OriginalDataStr:      cdmaGlobalState.OriginalDataStrB,
		EncodedDataStr:       cdmaGlobalState.EncodedDataStrB,
		DataLength:           cdmaGlobalState.DataLengthB,
		TransmittedSignalStr: cdmaGlobalState.TransmittedSignalBStr, // Populate added field
	}
	tmpl, err := template.ParseFiles("templates/cdma_transmitter_user_result.html")
	if err != nil {
		log.Printf("CDMATransmitterBResultsHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMATransmitterBResultsHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// CDMACodeAnalysisHandler returns code analysis results for CDMA
func CDMACodeAnalysisHandler(w http.ResponseWriter, r *http.Request) {
	cdmaGlobalState.mutex.RLock()
	defer cdmaGlobalState.mutex.RUnlock()
	if cdmaGlobalState.Timestamp == "" {
		http.Error(w, "Uruchom symulację CDMA.", http.StatusBadRequest)
		return
	}

	data := CDMACodeAnalysisData{
		Timestamp:                  cdmaGlobalState.Timestamp,
		AutocorrelationPeak:        cdmaGlobalState.AutocorrelationPeak,
		MaxOffPeakAutocorrelationA: cdmaGlobalState.MaxOffPeakAutocorrelationA,
		MaxOffPeakAutocorrelationB: cdmaGlobalState.MaxOffPeakAutocorrelationB,
		CrossCorrelationAB:         cdmaGlobalState.CrossCorrelationAB,
		GoldCodeLength:             cdmaGlobalState.GoldCodeLength,
	}

	tmpl, err := template.ParseFiles("templates/cdma_code_analysis_result.html")
	if err != nil {
		log.Printf("CDMACodeAnalysisHandler: Template error: %v", err)
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing CDMACodeAnalysisHandler template: %v", err)
		http.Error(w, "Template execution error", http.StatusInternalServerError)
	}
}

// Helper function to truncate string for display
func truncateString(s string, maxLength int) string {
	if len(s) <= maxLength {
		return s
	}
	if maxLength <= 3 { // Not enough space for "..."
		return s[:maxLength]
	}
	return s[:maxLength-3] + "..."
}

// Helper function to parse comma-separated taps
func parseTaps(tapsStr string) []uint {
	tapsStr = strings.TrimSpace(tapsStr)
	if tapsStr == "" {
		return nil
	}
	parts := strings.Split(tapsStr, ",")
	taps := make([]uint, 0, len(parts))
	for _, p := range parts {
		if val, err := strconv.Atoi(strings.TrimSpace(p)); err == nil {
			taps = append(taps, uint(val))
		}
	}
	return taps
}

// bitsToASCII converts a string of '0' and '1' to ASCII if length is a multiple of 8
func bitsToASCII(bits string) string {
	if len(bits)%8 != 0 || len(bits) == 0 {
		return ""
	}
	var sb strings.Builder
	for i := 0; i < len(bits); i += 8 {
		byteStr := bits[i : i+8]
		var b byte
		for j := 0; j < 8; j++ {
			b <<= 1
			if byteStr[j] == '1' {
				b |= 1
			}
		}
		sb.WriteByte(b)
	}
	return sb.String()
}

// --- Helper functions for parsing with defaults ---
func parseIntWithDefault(valStr string, defaultVal int, minVal int, maxVal int) int {
	if val, err := strconv.Atoi(strings.TrimSpace(valStr)); err == nil {
		if val >= minVal && val <= maxVal {
			return val
		}
	}
	return defaultVal
}

func parseUint64WithDefault(valStr string, defaultVal uint64) uint64 {
	trimmedValStr := strings.TrimSpace(valStr)
	if trimmedValStr == "" { // Handle empty string explicitly if needed, or rely on ParseUint failure
		return defaultVal
	}
	if val, err := strconv.ParseUint(trimmedValStr, 10, 64); err == nil {
		// It's common for seeds to be non-zero, but 0 can be a valid state for some LFSRs if handled.
		// For simplicity, let's allow 0 if parsed correctly.
		// If a strict non-zero policy is needed, add '&& val > 0'
		return val
	}
	return defaultVal
}

func parseFloatWithDefault(valStr string, defaultVal float64, minVal float64, maxVal float64) float64 {
	if val, err := strconv.ParseFloat(strings.TrimSpace(valStr), 64); err == nil {
		if val >= minVal && val <= maxVal {
			return val
		}
	}
	return defaultVal
}

func parseTapsWithDefault(tapsStr string, defaultTaps []uint) []uint {
	trimmedTapsStr := strings.TrimSpace(tapsStr)
	if trimmedTapsStr == "" {
		return defaultTaps
	}
	parsed := parseTaps(trimmedTapsStr) // parseTaps already handles trimming spaces around commas and parts
	if len(parsed) == 0 {
		// This case might occur if tapsStr is not empty but contains no valid numbers, e.g., "," or "a,b"
		return defaultTaps
	}
	return parsed
}
