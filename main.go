package main

import (
	"fmt"
	"log"
	"net/http"

	// Import the local src package
	"github.com/BartiX259/BSO_Projekt/src"
)

func main() {
	// --- Static File Server ---
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))
	log.Println("Serving static files from ./static/ directory under /static/ path")

	// --- Page Handlers ---
	http.HandleFunc("/", src.IndexHandler)
	http.HandleFunc("/simulate", src.SimulateHandler)
	http.HandleFunc("/download", src.DownloadResultsHandler)

	// --- Individual Module Handlers ---
	http.HandleFunc("/generator", src.GeneratorHandler)
	http.HandleFunc("/encoder", src.EncoderHandler)
	http.HandleFunc("/error", src.ErrorHandler)
	http.HandleFunc("/decoder", src.DecoderHandler)
	http.HandleFunc("/ber", src.BERHandler)
	http.HandleFunc("/autocorrelation", src.AutocorrelationHandler)

	// --- CDMA Simulation Handlers ---
	http.HandleFunc("/cdma-simulate", src.CDMASimulateHandler)
	http.HandleFunc("/cdma-system-config", src.CDMASystemConfigHandler)                // Module 1
	http.HandleFunc("/cdma-transmitter-a-results", src.CDMATransmitterAResultsHandler) // Module 2A
	http.HandleFunc("/cdma-transmitter-b-results", src.CDMATransmitterBResultsHandler) // Module 2B
	http.HandleFunc("/cdma-channel-results", src.CDMAChannelResultsHandler)            // Module 3
	http.HandleFunc("/cdma-receiver-a-results", src.CDMAReceiverAResultsHandler)       // Module 4A
	http.HandleFunc("/cdma-receiver-b-results", src.CDMAReceiverBResultsHandler)       // Module 4B
	http.HandleFunc("/cdma-ber-a-results", src.CDMABERAResultsHandler)                 // Module 5A
	http.HandleFunc("/cdma-ber-b-results", src.CDMABERBResultsHandler)                 // Module 5B
	http.HandleFunc("/cdma-code-analysis", src.CDMACodeAnalysisHandler)                // Module 6
	// --- END NEW ---

	// --- Start Server ---
	port := ":8080"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
