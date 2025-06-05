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

	// --- Start Server ---
	port := ":8080"
	fmt.Printf("Server starting on http://localhost%s\n", port)
	log.Fatal(http.ListenAndServe(port, nil))
}
