package src

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
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
	DecodedASCII    string // NEW: ASCII representation
}

// BERData holds data for BER template
type BERData struct {
	BER              string
	ErrorsDetected   int
	TotalBits        int
	OriginalSequence string
	DecodedSequence  string
	OriginalASCII    string // NEW: ASCII representation of original
	DecodedASCII     string // NEW: ASCII representation of decoded
}

// AutocorrelationData holds data for autocorrelation template
type AutocorrelationData struct {
	OriginalMaxOffPeak  string
	EncodedMaxOffPeak   string
	CorruptedMaxOffPeak string
}

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
	seqText := strings.TrimSpace(r.FormValue("seqText"))
	seqLengthStr := strings.TrimSpace(r.FormValue("seqLength"))
	errorRateStr := strings.TrimSpace(r.FormValue("errorRate"))
	errorType := strings.TrimSpace(r.FormValue("errorType"))
	goldNStr := strings.TrimSpace(r.FormValue("goldN"))
	goldTaps1Str := strings.TrimSpace(r.FormValue("goldTaps1"))
	goldTaps2Str := strings.TrimSpace(r.FormValue("goldTaps2"))
	decoderType := strings.TrimSpace(r.FormValue("decoderType"))

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
	if seqText != "" {
		bitSeq = simulation.StringAsSequence(seqText)
		log.Printf("Using text input: '%s'", seqText)
	} else {
		seqLength := 64
		if seqLengthStr != "" {
			if parsed, err := strconv.Atoi(seqLengthStr); err == nil && parsed > 0 {
				seqLength = parsed
			}
		}
		bitSeq = simulation.RandomSequence(seqLength)
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

	// Run complete simulation pipeline
	encoded := simulation.EncodeWithGold(*bitSeq, *goldCode)
	log.Printf("Encoded sequence length: %d", encoded.Len())

	// Convert error rate from percentage to decimal for the function
	errorRateDecimal := errorRate / 100.0
	corrupted, errorsIntroduced := simulation.AddErrors(encoded, errorRateDecimal, errorType)
	log.Printf("Errors introduced: %d (rate: %.4f)", errorsIntroduced, errorRateDecimal)

	decoded := simulation.DecodeWithGold(*corrupted, *goldCode)
	ber := simulation.CalculateBER(*bitSeq, *decoded)

	// Calculate autocorrelation analysis
	originalAutocorr := simulation.MaxAbsoluteOffPeak(simulation.CalculatePeriodicAutocorrelation(*bitSeq))
	encodedAutocorr := simulation.MaxAbsoluteOffPeak(simulation.CalculatePeriodicAutocorrelation(*encoded))
	corruptedAutocorr := simulation.MaxAbsoluteOffPeak(simulation.CalculatePeriodicAutocorrelation(*corrupted))

	// Count errors for display
	errorCount := 0
	for i := range bitSeq.Len() {
		if bitSeq.Get(i) != decoded.Get(i) {
			errorCount++
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
	globalResults.InputText = seqText
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

	// Get stored results from global state
	globalResults.mutex.RLock()

	// Check if we have stored results
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

	// Get stored results from global state
	globalResults.mutex.RLock()

	// Check if we have stored results
	if globalResults.Decoded == nil {
		globalResults.mutex.RUnlock()
		log.Printf("DecoderHandler: No decoded sequence available")
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
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
	// Get stored results from global state
	globalResults.mutex.RLock()

	// Check if we have stored results
	if globalResults.Original == nil || globalResults.Decoded == nil {
		globalResults.mutex.RUnlock()
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
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

// Helper function to parse comma-separated taps
func parseTaps(tapsStr string) []uint {
	if tapsStr == "" {
		return nil
	}

	parts := strings.Split(tapsStr, ",")
	taps := make([]uint, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if tap, err := strconv.Atoi(part); err == nil && tap >= 0 {
			taps = append(taps, uint(tap))
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
