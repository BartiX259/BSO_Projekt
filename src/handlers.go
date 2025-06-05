package src

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
	"github.com/BartiX259/BSO_Projekt/src/simulation"
)

// Global simulation results storage
type SimulationResults struct {
	Original    *simulation.BitSequence
	GoldCode    *simulation.BitSequence
	Encoded     *simulation.BitSequence
	Corrupted   *simulation.BitSequence
	Decoded     *simulation.BitSequence
	BER         float32
	ErrorCount  int
	InputText   string
	ErrorType   string
	ErrorRate   float64
	ErrorsIntroduced int
	Timestamp   string
	mutex       sync.RWMutex
}

var globalResults = &SimulationResults{}

// ResponseData holds data for the response template
type ResponseData struct {
	Timestamp   	string
	BitSequence 	string
	EncodedSequence	string
	DecodedSequence string
	BER				string
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
}

// ErrorData holds data for error template
type ErrorData struct {
	CorruptedSequence  string
	ErrorType          string
	ErrorRate          float64
	ErrorsIntroduced   int
}

// DecoderData holds data for decoder template
type DecoderData struct {
	DecodedSequence string
	DecoderType     string
}

// BERData holds data for BER template
type BERData struct {
	BER              string
	ErrorsDetected   int
	TotalBits        int
	OriginalSequence string
	DecodedSequence  string
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
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	// Parse form data
	seqText := r.FormValue("seqText")
	seqLengthStr := r.FormValue("seqLength")
	errorRateStr := r.FormValue("errorRate")
	errorType := r.FormValue("errorType")
	
	// Generate or parse bit sequence
	var bitSeq *simulation.BitSequence
	if seqText != "" {
		bitSeq = simulation.StringAsSequence(seqText)
	} else {
		seqLength := 64
		if seqLengthStr != "" {
			if parsed, err := strconv.Atoi(seqLengthStr); err == nil && parsed > 0 {
				seqLength = parsed
			}
		}
		bitSeq = simulation.RandomSequence(seqLength)
	}

	// Parse error parameters
	errorRate := 5.0
	if errorRateStr != "" {
		if parsed, err := strconv.ParseFloat(errorRateStr, 64); err == nil && parsed >= 0 && parsed <= 100 {
			errorRate = parsed
		}
	}
	
	if errorType == "" {
		errorType = "random"
	}

	// Generate Gold code
	n := 10
	taps1 := []uint{0, 3}
	taps2 := []uint{0, 2, 3, 8}
	seed1 := uint64(1)
	seed2 := uint64(0b1010101010)
	goldCode := simulation.GenerateGoldCode(uint(n), taps1, seed1, taps2, seed2)
	
	// Run complete simulation pipeline
	encoded := simulation.EncodeWithGold(*bitSeq, *goldCode)
	corrupted, errorsIntroduced := simulation.AddErrors(encoded, errorRate, errorType)
	decoded := simulation.DecodeWithGold(*corrupted, *goldCode)
	ber := simulation.CalculateBER(*bitSeq, *decoded)
	
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
	globalResults.mutex.Unlock()

	// Prepare response data
	data := ResponseData{
		Timestamp:       globalResults.Timestamp,
		BitSequence:     bitSeq.String(),
		EncodedSequence: encoded.String(),
		DecodedSequence: decoded.String(),
		BER:            fmt.Sprintf("%.2f", ber*100),
	}

	tmpl, err := template.ParseFiles("templates/response.html")
	if err != nil {
		log.Printf("Error parsing response template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing response template: %v", err)
	}
}

// GeneratorHandler returns stored bit sequence generation results
func GeneratorHandler(w http.ResponseWriter, r *http.Request) {
	// Get stored results from global state
	globalResults.mutex.RLock()
	
	// Check if we have stored results
	if globalResults.Original == nil {
		globalResults.mutex.RUnlock()
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}
	
	data := GeneratorData{
		BitSequence: globalResults.Original.String(),
		InputText:   globalResults.InputText,
		Length:      globalResults.Original.Len(),
	}
	globalResults.mutex.RUnlock()

	tmpl, err := template.ParseFiles("templates/generator_result.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing generator template: %v", err)
	}
}

// EncoderHandler returns stored Gold code generation and encoding results
func EncoderHandler(w http.ResponseWriter, r *http.Request) {
	// Get stored results from global state
	globalResults.mutex.RLock()
	
	// Check if we have stored results
	if globalResults.GoldCode == nil || globalResults.Encoded == nil {
		globalResults.mutex.RUnlock()
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}

	data := EncoderData{
		GoldCode:        globalResults.GoldCode.String(),
		EncodedSequence: globalResults.Encoded.String(),
		N:               10, // Fixed value as used in simulation
		Length:          globalResults.GoldCode.Len(),
	}
	globalResults.mutex.RUnlock()

	tmpl, err := template.ParseFiles("templates/encoder_result.html")
	if err != nil {
		http.Error(w, "Template error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("Error executing encoder template: %v", err)
	}
}

// ErrorHandler returns stored error injection results
func ErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Get stored results from global state
	globalResults.mutex.RLock()
	
	// Check if we have stored results
	if globalResults.Corrupted == nil {
		globalResults.mutex.RUnlock()
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

	tmpl, err := template.ParseFiles("templates/error_result.html")
	if err != nil {
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
	// Get stored results from global state
	globalResults.mutex.RLock()
	
	// Check if we have stored results
	if globalResults.Decoded == nil {
		globalResults.mutex.RUnlock()
		http.Error(w, "No simulation results available. Please run complete simulation first.", http.StatusBadRequest)
		return
	}
	
	data := DecoderData{
		DecodedSequence: globalResults.Decoded.String(),
		DecoderType:     "xor",
	}
	globalResults.mutex.RUnlock()

	tmpl, err := template.ParseFiles("templates/decoder_result.html")
	if err != nil {
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

	data := BERData{
		BER:              fmt.Sprintf("%.2f", globalResults.BER*100),
		ErrorsDetected:   globalResults.ErrorCount,
		TotalBits:        globalResults.Original.Len(),
		OriginalSequence: globalResults.Original.String(),
		DecodedSequence:  globalResults.Decoded.String(),
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