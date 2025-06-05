package src

import (
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/BartiX259/BSO_Projekt/src/simulation"
)

// ResponseData holds data for the response template (Exported)
type ResponseData struct {
	Timestamp   string
	BitSequence string
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

// ClickedHandler handles the form
func ClickedHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Form parse error", http.StatusBadRequest)
		return
	}

	input := r.FormValue("inputString")
	var bitSeq string
	if input != "" {
		bitSeq = simulation.StringAsSequence(input).String()
	} else {
		bitSeq = simulation.RandomSequence(64).String()
	}

	data := ResponseData{
		Timestamp:   time.Now().Format(time.RFC1123),
		BitSequence: bitSeq,
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
