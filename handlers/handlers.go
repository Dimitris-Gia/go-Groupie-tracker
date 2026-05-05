// Package handlers contains all HTTP handler functions and supporting types
// for the Groupie Tracker web application.
package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"groupie-tracker/api"
)

// ArtistPageData holds all data passed to the artist detail template.
type ArtistPageData struct {
	Artist    *api.Artist
	Locations *api.Locations
	Dates     *api.Dates
	Relations *api.Relations
}

// SearchSuggestion represents a single autocomplete suggestion item.
type SearchSuggestion struct {
	Text string `json:"text"`
	Type string `json:"type"` // e.g. "artist/band", "member", "location"
}

// SearchResponse is the JSON payload returned by the /search endpoint.
type SearchResponse struct {
	Results     []api.Artist       `json:"results"`
	Suggestions []SearchSuggestion `json:"suggestions"`
}

// renderError writes an HTTP error response using the error.html template.
// Falls back to a plain text response if the template cannot be parsed or executed.
func renderError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	tmpl, err := template.ParseFiles("templates/error.html")
	if err != nil {
		w.Write([]byte(message))
		return
	}
	data := struct {
		Code    int
		Message string
	}{Code: status, Message: message}

	if err := tmpl.Execute(w, data); err != nil {
		w.Write([]byte(message))
	}
}

// HomeHandler serves the main page at "/".
// It fetches all artists and locations, applies filters and pagination, then renders index.html.
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Reject any path that is not exactly "/"
	if r.URL.Path != "/" {
		renderError(w, http.StatusNotFound, "Page not found")
		return
	}
	tmpl, err := template.New("index.html").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"gt":  func(a, b int) bool { return a > b },
		"lt":  func(a, b int) bool { return a < b },
		// seq generates a slice of integers from start to end (inclusive), used for pagination links
		"seq": func(start, end int) []int {
			nums := make([]int, end-start+1)
			for i := range nums {
				nums[i] = start + i
			}
			return nums
		},
	}).ParseFiles("templates/index.html")
	if err != nil {
		renderError(w, http.StatusNotFound, "Template not found")
		return
	}

	artists, err := api.Api("https://groupietrackers.herokuapp.com/api/artists")
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Could not load artists")
		return
	}

	allLocationsData, err := api.GetAllLocations()
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Could not load locations")
		return
	}

	// Build a map of artist ID → locations slice for O(1) lookup in the filter
	locMap := make(map[int][]string)
	for _, loc := range allLocationsData {
		locMap[loc.Id] = loc.Locations
	}

	// Build a sorted, deduplicated list of all unique locations for the dropdown.
	// Locations are stored as "city-country"; convert underscores to spaces for display.
	seen := map[string]bool{}
	var allLocs []string
	for _, loc := range allLocationsData {
		for _, l := range loc.Locations {
			display := strings.ReplaceAll(l, "_", " ")
			if !seen[display] {
				seen[display] = true
				allLocs = append(allLocs, display)
			}
		}
	}
	sort.Strings(allLocs)

	// Apply query-string filters (year range, album range, member count, location)
	filtered := Filters(artists, locMap, r)
	// Slice the filtered list into the requested page
	data := Pagination(6, filtered, allLocs, r)

	if err = tmpl.Execute(w, data); err != nil {
		renderError(w, http.StatusInternalServerError, "Internal server error")
	}
}

// ArtistHandler serves the artist detail page at "/artist/{id}".
// It fetches the artist, their locations, dates, and relations, then renders artist.html.
// An optional "tab" query param (dates|locations|relations) controls which section is shown.
func ArtistHandler(w http.ResponseWriter, r *http.Request) {
	// Lightweight JSON endpoint used by the map UI.
	// Path: /artist/{id}/coords
	if strings.HasSuffix(r.URL.Path, "/coords") {
		idStr := strings.TrimPrefix(r.URL.Path, "/artist/")
		idStr = strings.TrimSuffix(idStr, "/coords")
		idStr = strings.Trim(idStr, "/")
		id, err := strconv.Atoi(idStr)
		if err != nil || id < 1 {
			http.Error(w, "Invalid artist ID", http.StatusBadRequest)
			return
		}

		locations, err := api.GetLocations(id)
		if err != nil {
			http.Error(w, "Could not load locations", http.StatusInternalServerError)
			return
		}

		coords := make([]api.GeoCoord, 0, len(locations.Locations))
		for _, loc := range locations.Locations {
			if coord, err := api.Geocode(loc); err == nil {
				coords = append(coords, coord)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(coords)
		return
	}

	// Extract the numeric ID from the URL path
	idStr := strings.TrimPrefix(r.URL.Path, "/artist/")
	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		renderError(w, http.StatusBadRequest, "Invalid artist ID")
		return
	}

	artist, err := api.GetArtist(id)
	if err != nil {
		renderError(w, http.StatusNotFound, "Artist not found")
		return
	}
	locations, err := api.GetLocations(id)
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Could not load locations")
		return
	}
	dates, err := api.GetDates(id)
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Could not load dates")
		return
	}
	for i := range dates.Dates {
		raw := strings.TrimSpace(dates.Dates[i])
		dates.Dates[i] = strings.TrimPrefix(raw, "*")
	}
	relations, err := api.GetRelations(id)
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Could not load relations")
		return
	}

	tmpl, err := template.ParseFiles("templates/artist.html")
	if err != nil {
		renderError(w, http.StatusInternalServerError, "Template not found")
		return
	}
	tmpl.Execute(w, ArtistPageData{Artist: artist, Locations: locations, Dates: dates, Relations: relations})
}

// The seen map prevents duplicate suggestions across different match types.
func addSuggestion(suggestions *[]SearchSuggestion, seen map[string]bool, text, typ string) {
	key := text + "|" + typ
	if text == "" || seen[key] {
		return
	}
	seen[key] = true
	*suggestions = append(*suggestions, SearchSuggestion{Text: text, Type: typ})
}

// SearchHandler handles live search requests at "/search?q=...".
// It matches the query against artist names, members, locations, first album dates,
// and creation dates, returning a JSON response with results and autocomplete suggestions.
func SearchHandler(w http.ResponseWriter, r *http.Request) {
	rawQuery := strings.TrimSpace(r.URL.Query().Get("q"))
	query := strings.ToLower(rawQuery)

	artists, err := api.Api("https://groupietrackers.herokuapp.com/api/artists")
	if err != nil {
		http.Error(w, "API error", http.StatusInternalServerError)
		return
	}

	// Fetch all locations to enable location-based search
	allLocations, err := api.GetAllLocations()
	if err != nil {
		http.Error(w, "API error", http.StatusInternalServerError)
		return
	}

	// Build a map of artist ID → locations for O(1) lookup during search
	locMap := make(map[int][]string)
	for _, loc := range allLocations {
		locMap[loc.Id] = loc.Locations
	}

	response := SearchResponse{Results: artists}
	if query != "" {
		response.Results = nil
		seen := map[string]bool{}

		for _, art := range artists {
			matched := false

			if strings.Contains(strings.ToLower(art.Name), query) {
				matched = true
				addSuggestion(&response.Suggestions, seen, art.Name, "artist/band")
			}

			for _, member := range art.Members {
				if strings.Contains(strings.ToLower(member), query) {
					matched = true
					addSuggestion(&response.Suggestions, seen, member, "member")
				}
			}

			if locs, ok := locMap[art.Id]; ok {
				for _, loc := range locs {
					// Replace underscores with spaces for display (e.g. "new_york" → "new york")
					displayLoc := strings.ReplaceAll(loc, "_", " ")
					if strings.Contains(strings.ToLower(displayLoc), query) {
						matched = true
						addSuggestion(&response.Suggestions, seen, displayLoc, "location")
					}
				}
			}

			if strings.Contains(strings.ToLower(art.FirstAlbum), query) {
				matched = true
				addSuggestion(&response.Suggestions, seen, art.FirstAlbum, "first album date")
			}

			if strings.Contains(strconv.Itoa(art.CreationDate), query) {
				matched = true
				addSuggestion(&response.Suggestions, seen, strconv.Itoa(art.CreationDate), "creation date")
			}

			if matched {
				response.Results = append(response.Results, art)
			}
		}
	}

	// Sort suggestions: entries that start with the query come before those that only contain it
	sort.SliceStable(response.Suggestions, func(i, j int) bool {
		iStarts := strings.HasPrefix(strings.ToLower(response.Suggestions[i].Text), query)
		jStarts := strings.HasPrefix(strings.ToLower(response.Suggestions[j].Text), query)
		if iStarts != jStarts {
			return iStarts
		}
		return false
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
