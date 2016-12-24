package dependevent

import (
	"html/template"
	"net/http"
)

var (
	pages *template.Template
)

func init() {
	pages = template.Must(template.ParseGlob("templates/*.template"))

	http.HandleFunc("/edit_event", editEvent)
	http.HandleFunc("/save_event", saveEvent)
	http.HandleFunc("/complete_event", completeEvent)

	http.HandleFunc("/dashboard", dashboard)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/", index)
}
