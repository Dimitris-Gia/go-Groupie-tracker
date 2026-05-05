package main

// Groupie Tracker - main entry point.
// Starts an HTTP server on port 8080, serves static files and registers route handlers.
// External API base URL: https://groupietrackers.herokuapp.com/api/artists

import (
	"log"
	"net/http"

	"groupie-tracker/api"
	"groupie-tracker/handlers"
)

func main() {
	// Serve files from the /static/ directory (CSS, JS, images)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Home page: artist grid with filters and pagination
	http.HandleFunc("/", handlers.HomeHandler)
	// Artist detail page: /artist/{id} with optional ?tab=dates|locations|relations
	http.HandleFunc("/artist/", handlers.ArtistHandler)
	// Live search endpoint: returns JSON results and suggestions
	http.HandleFunc("/search", handlers.SearchHandler)

	// Pre-warm the geocode cache in the background so map pages load instantly
	go api.PrewarmGeocodeCache()

	log.Println("Server starting on http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
