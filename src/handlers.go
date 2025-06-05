package src

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/BartiX259/BSO_Projekt/src/simulation"
)

// ResponseData holds data for the response template
type ResponseData struct {
	Timestamp   	string
	BitSequence 	string
	EncodedSequence	string
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

// Handle the simulation endpoint
func SimulateHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	formBitSequence := r.FormValue("bitSequence")
	var bitSeq *simulation.BitSequence
	if formBitSequence != "" {
		bitSeq = simulation.StringAsSequence(formBitSequence)
	} else {
		bitSeq = simulation.RandomSequence(64)
	}

	n := 10
	taps1 := []uint{2, 9}   // For example: bits 3 and 10
	taps2 := []uint{2, 3, 6, 8, 9} // More complex second LFSR
	seed1 := uint64(0b1000000001) // any non-zero 10-bit value
	seed2 := uint64(0b1101011101)

	goldCode := simulation.GenerateGoldCode(uint(n), taps1, seed1, taps2, seed2)
	log.Printf("gold code: %s", goldCode.String())
	encoded := simulation.EncodeWithGold(*bitSeq, *goldCode)
	log.Printf("encoded: %s", encoded.String())

	data := ResponseData{
		Timestamp:   time.Now().Format(time.RFC1123),
		BitSequence: bitSeq.String(),
		EncodedSequence: encoded.String(),
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
