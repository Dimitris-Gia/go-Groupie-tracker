// Package api handles all communication with the Groupie Trackers external REST API.
// Base URL: https://groupietrackers.herokuapp.com/api
package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Artist represents a music artist or band returned by the /api/artists endpoint.
type Artist struct {
	Id           int      `json:"id"`
	Image        string   `json:"image"`
	Name         string   `json:"name"`
	Members      []string `json:"members"`
	CreationDate int      `json:"creationDate"`
	FirstAlbum   string   `json:"firstAlbum"`
	Locations    string   `json:"locations"`    // URL to the artist's locations endpoint
	ConcertDates string   `json:"concertDates"` // URL to the artist's dates endpoint
	Relations    string   `json:"relations"`    // URL to the artist's relations endpoint
}

// Locations holds the concert locations for a single artist.
type Locations struct {
	Id        int      `json:"id"`
	Locations []string `json:"locations"`
	Dates     string   `json:"dates"`
}

// AllLocations wraps the index array returned by /api/locations.
type AllLocations struct {
	Index []Locations `json:"index"`
}

// Dates holds the concert dates for a single artist.
type Dates struct {
	Id    int      `json:"id"`
	Dates []string `json:"dates"`
}

// Relations maps each concert location to its list of dates for a single artist.
type Relations struct {
	Id             int                 `json:"id"`
	DatesLocations map[string][]string `json:"datesLocations"`
}

// Api fetches the full list of artists from the given URL.
func Api(url string) ([]Artist, error) {
	var artists []Artist
	err := fetchJSON(url, &artists)
	return artists, err
}

// fetchJSON is a shared helper that performs a GET request and unmarshals the JSON body into target.
func fetchJSON(url string, target interface{}) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, target)
}

// GetArtist fetches a single artist by ID from /api/artists/{id}.
func GetArtist(id int) (*Artist, error) {
	var artist Artist
	err := fetchJSON(fmt.Sprintf("https://groupietrackers.herokuapp.com/api/artists/%d", id), &artist)
	return &artist, err
}

// GetLocations fetches the concert locations for a single artist by ID.
func GetLocations(id int) (*Locations, error) {
	var loc Locations
	err := fetchJSON(fmt.Sprintf("https://groupietrackers.herokuapp.com/api/locations/%d", id), &loc)
	return &loc, err
}

// GetDates fetches the concert dates for a single artist by ID.
func GetDates(id int) (*Dates, error) {
	var dates Dates
	err := fetchJSON(fmt.Sprintf("https://groupietrackers.herokuapp.com/api/dates/%d", id), &dates)
	return &dates, err
}

// GetRelations fetches the dates-locations mapping for a single artist by ID.
func GetRelations(id int) (*Relations, error) {
	var rel Relations
	err := fetchJSON(fmt.Sprintf("https://groupietrackers.herokuapp.com/api/relation/%d", id), &rel)
	return &rel, err
}

// GetAllLocations fetches the full locations list for all artists from /api/locations.
func GetAllLocations() ([]Locations, error) {
	var all AllLocations
	err := fetchJSON("https://groupietrackers.herokuapp.com/api/locations", &all)
	return all.Index, err
}

// GeoCoord holds a geocoded location with its display name and coordinates.
type GeoCoord struct {
	Name string  `json:"name"`
	Lat  float64 `json:"lat"`
	Lng  float64 `json:"lng"`
}

// nominatimResult maps the fields we need from a Nominatim JSON response.
type nominatimResult struct {
	Lat     string `json:"lat"`
	Lon     string `json:"lon"`
	Display string `json:"display_name"`
}

var (
	geocodeCache    = map[string]GeoCoord{}
	geocodeCacheMu  sync.RWMutex
	lastRequestTime time.Time
	requestMu       sync.Mutex
)

// PrewarmGeocodeCache fetches all artist locations and geocodes each unique one sequentially.
// It is intended to be called once as a goroutine at server startup so that by the time
// a user opens an artist map page all coordinates are already in the in-memory cache.
func PrewarmGeocodeCache() {
	allLocs, err := GetAllLocations()
	if err != nil {
		return
	}
	seen := map[string]bool{}
	for _, artistLoc := range allLocs {
		for _, loc := range artistLoc.Locations {
			if seen[loc] {
				continue
			}
			seen[loc] = true
			Geocode(loc) // result is cached internally; error is intentionally ignored
		}
	}
}

// Geocode converts a raw location string (e.g. "north_carolina-usa") to a GeoCoord.
// It cleans the string, queries the Nominatim API, and returns the first result.
// Results are cached to avoid repeated API calls and respect rate limits.
func Geocode(location string) (GeoCoord, error) {
	// Check cache first
	geocodeCacheMu.RLock()
	if coord, ok := geocodeCache[location]; ok {
		geocodeCacheMu.RUnlock()
		return coord, nil
	}
	geocodeCacheMu.RUnlock()

	// Rate limit: ensure at least 1 second between requests
	requestMu.Lock()
	if elapsed := time.Since(lastRequestTime); elapsed < time.Second {
		time.Sleep(time.Second - elapsed)
	}
	lastRequestTime = time.Now()
	requestMu.Unlock()

	// Convert API format "city-country" to "city, country" for better geocoding
	// Replace underscores with spaces, then replace the dash with comma+space
	query := strings.ReplaceAll(location, "_", " ")
	if idx := strings.LastIndex(query, "-"); idx != -1 {
		query = query[:idx] + ", " + query[idx+1:]
	}
	url := "https://nominatim.openstreetmap.org/search?q=" + neturl.QueryEscape(query) + "&format=json&limit=1"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return GeoCoord{}, err
	}
	// Nominatim requires a User-Agent header
	req.Header.Set("User-Agent", "groupie-tracker/1.0")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return GeoCoord{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return GeoCoord{}, err
	}

	var results []nominatimResult
	if err := json.Unmarshal(body, &results); err != nil || len(results) == 0 {
		return GeoCoord{}, fmt.Errorf("no results for %q", query)
	}

	lat, _ := strconv.ParseFloat(results[0].Lat, 64)
	lng, _ := strconv.ParseFloat(results[0].Lon, 64)
	coord := GeoCoord{Name: query, Lat: lat, Lng: lng}

	// Cache the result
	geocodeCacheMu.Lock()
	geocodeCache[location] = coord
	geocodeCacheMu.Unlock()

	return coord, nil
}
