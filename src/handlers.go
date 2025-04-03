// src/handlers.go
package src // Package name matches the directory

import (
	"html/template"
	"log"
	"net/http"
	"time"
)

// ResponseData holds data for the response template (Exported)
type ResponseData struct {
	Timestamp string
}

// IndexHandler serves the main HTML page using a template (Exported)
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

// ClickedHandler handles the HTMX request, rendering a template fragment (Exported)
func ClickedHandler(w http.ResponseWriter, r *http.Request) {
	time.Sleep(300 * time.Millisecond) // Simulate work

	currentTime := time.Now().Format(time.RFC1123)
	data := ResponseData{ // Use the exported struct name
		Timestamp: currentTime,
	}

	tmpl, err := template.ParseFiles("templates/response.html") // Path relative to execution dir
	if err != nil {
		log.Printf("Error parsing response template: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html")
	err = tmpl.Execute(w, data)
	if err != nil {
		log.Printf("Error executing response template: %v", err)
		return
	}
	log.Println("Served /clicked response using template from src package.")
}