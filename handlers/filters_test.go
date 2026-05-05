package handlers

import (
	"net/http"
	"net/url"
	"testing"

	"groupie-tracker/api"
)

// makeRequest builds a GET *http.Request with the given query parameters.
func makeRequest(params map[string]string) *http.Request {
	v := url.Values{}
	for k, val := range params {
		v.Set(k, val)
	}
	r, _ := http.NewRequest("GET", "/?"+v.Encode(), nil)
	return r
}

// makeRequestMulti builds a GET *http.Request supporting multi-value params (e.g. members).
func makeRequestMulti(params url.Values) *http.Request {
	r, _ := http.NewRequest("GET", "/?"+params.Encode(), nil)
	return r
}

var testArtists = []api.Artist{
	{Id: 1, Name: "Queen", Members: []string{"Freddie", "Brian", "Roger", "John"}, CreationDate: 1970, FirstAlbum: "13-07-1973"},
	{Id: 2, Name: "Metallica", Members: []string{"James", "Lars", "Kirk", "Robert"}, CreationDate: 1981, FirstAlbum: "27-07-1983"},
	{Id: 3, Name: "Muse", Members: []string{"Matt", "Chris", "Dom"}, CreationDate: 1994, FirstAlbum: "11-10-1999"},
	{Id: 4, Name: "Eminem", Members: []string{"Eminem"}, CreationDate: 1996, FirstAlbum: "12-02-1999"},
}

// testLocMap provides concert locations for each test artist.
var testLocMap = map[int][]string{
	1: {"london-uk", "paris-france"},
	2: {"seattle-washington-usa", "new_york-usa"},
	3: {"london-uk", "berlin-germany"},
	4: {"los_angeles-usa", "chicago-usa"},
}

func TestFilters_NoFilters_ReturnsAll(t *testing.T) {
	r := makeRequest(map[string]string{})
	result := Filters(testArtists, testLocMap, r)
	if len(result) != len(testArtists) {
		t.Errorf("expected %d artists, got %d", len(testArtists), len(result))
	}
}

func TestFilters_YearRange_FiltersCorrectly(t *testing.T) {
	r := makeRequest(map[string]string{"yearFrom": "1980", "yearTo": "1990"})
	result := Filters(testArtists, testLocMap, r)
	// Only Metallica (CreationDate: 1981) falls in 1980–1990
	if len(result) != 1 || result[0].Name != "Metallica" {
		t.Errorf("expected only Metallica, got %+v", result)
	}
}

func TestFilters_YearRange_SwappedBounds(t *testing.T) {
	r := makeRequest(map[string]string{"yearFrom": "1990", "yearTo": "1980"})
	result := Filters(testArtists, testLocMap, r)
	// Swapped bounds should still match Metallica (CreationDate: 1981)
	if len(result) != 1 || result[0].Name != "Metallica" {
		t.Errorf("expected only Metallica, got %+v", result)
	}
}

func TestFilters_AlbumRange_FiltersCorrectly(t *testing.T) {
	r := makeRequest(map[string]string{"albumFrom": "1995", "albumTo": "2000"})
	result := Filters(testArtists, testLocMap, r)
	// Muse (1999) and Eminem (1999) fall in 1995–2000
	if len(result) != 2 {
		t.Errorf("expected 2 artists, got %d: %+v", len(result), result)
	}
}

func TestFilters_Members_SingleValue(t *testing.T) {
	v := url.Values{}
	v.Add("members", "3")
	r := makeRequestMulti(v)
	result := Filters(testArtists, testLocMap, r)
	if len(result) != 1 || result[0].Name != "Muse" {
		t.Errorf("expected only Muse, got %+v", result)
	}
}

func TestFilters_Members_MultipleValues(t *testing.T) {
	v := url.Values{}
	v.Add("members", "1")
	v.Add("members", "3")
	r := makeRequestMulti(v)
	result := Filters(testArtists, testLocMap, r)
	// Muse (3) and Eminem (1)
	if len(result) != 2 {
		t.Errorf("expected 2 artists, got %d: %+v", len(result), result)
	}
}

func TestFilters_InvalidYearParams_SkipsArtist(t *testing.T) {
	r := makeRequest(map[string]string{"yearFrom": "abc", "yearTo": "2000"})
	result := Filters(testArtists, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists, got %d", len(result))
	}
}

func TestFilters_EmptyArtistList(t *testing.T) {
	r := makeRequest(map[string]string{"yearFrom": "1970", "yearTo": "2000"})
	result := Filters([]api.Artist{}, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists, got %d", len(result))
	}
}

func TestFilters_Location_ExactCity(t *testing.T) {
	r := makeRequest(map[string]string{"location": "london"})
	result := Filters(testArtists, testLocMap, r)
	// Queen and Muse both have london-uk
	if len(result) != 2 {
		t.Errorf("expected 2 artists for 'london', got %d: %+v", len(result), result)
	}
}

func TestFilters_Location_SubstringMatchesRegion(t *testing.T) {
	// "washington" should match "seattle-washington-usa" (the hint case)
	r := makeRequest(map[string]string{"location": "washington"})
	result := Filters(testArtists, testLocMap, r)
	if len(result) != 1 || result[0].Name != "Metallica" {
		t.Errorf("expected only Metallica for 'washington', got %+v", result)
	}
}

func TestFilters_Location_NoMatch(t *testing.T) {
	r := makeRequest(map[string]string{"location": "tokyo"})
	result := Filters(testArtists, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists for 'tokyo', got %d", len(result))
	}
}

func TestFilters_Location_UnderscoreNormalised(t *testing.T) {
	// "los angeles" (with space) should match "los_angeles-usa"
	r := makeRequest(map[string]string{"location": "los angeles"})
	result := Filters(testArtists, testLocMap, r)
	if len(result) != 1 || result[0].Name != "Eminem" {
		t.Errorf("expected only Eminem for 'los angeles', got %+v", result)
	}
}

func TestFilters_YearFrom_OnlyLowerBound(t *testing.T) {
	r := makeRequest(map[string]string{"yearFrom": "1980"})
	result := Filters(testArtists, testLocMap, r)
	// yearFrom without yearTo should not filter
	if len(result) != len(testArtists) {
		t.Errorf("expected all artists when only yearFrom is set, got %d", len(result))
	}
}

func TestFilters_YearTo_OnlyUpperBound(t *testing.T) {
	r := makeRequest(map[string]string{"yearTo": "1990"})
	result := Filters(testArtists, testLocMap, r)
	// yearTo without yearFrom should not filter
	if len(result) != len(testArtists) {
		t.Errorf("expected all artists when only yearTo is set, got %d", len(result))
	}
}

func TestFilters_AlbumFrom_OnlyLowerBound(t *testing.T) {
	r := makeRequest(map[string]string{"albumFrom": "1995"})
	result := Filters(testArtists, testLocMap, r)
	// albumFrom without albumTo should not filter
	if len(result) != len(testArtists) {
		t.Errorf("expected all artists when only albumFrom is set, got %d", len(result))
	}
}

func TestFilters_AlbumTo_OnlyUpperBound(t *testing.T) {
	r := makeRequest(map[string]string{"albumTo": "2000"})
	result := Filters(testArtists, testLocMap, r)
	// albumTo without albumFrom should not filter
	if len(result) != len(testArtists) {
		t.Errorf("expected all artists when only albumTo is set, got %d", len(result))
	}
}

func TestFilters_InvalidAlbumParams_SkipsArtist(t *testing.T) {
	r := makeRequest(map[string]string{"albumFrom": "abc", "albumTo": "2000"})
	result := Filters(testArtists, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists for invalid album params, got %d", len(result))
	}
}

func TestFilters_ShortFirstAlbum_SkipsArtist(t *testing.T) {
	// Artist with a malformed FirstAlbum field (less than 4 chars)
	artists := []api.Artist{
		{Id: 1, Name: "BadData", FirstAlbum: "99", CreationDate: 1990, Members: []string{"A"}},
	}
	r := makeRequest(map[string]string{"albumFrom": "1990", "albumTo": "2000"})
	result := Filters(artists, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists for short FirstAlbum, got %d", len(result))
	}
}

func TestFilters_InvalidAlbumYear_SkipsArtist(t *testing.T) {
	// Artist with non-numeric year in FirstAlbum
	artists := []api.Artist{
		{Id: 1, Name: "BadYear", FirstAlbum: "12-05-ABCD", CreationDate: 1990, Members: []string{"A"}},
	}
	r := makeRequest(map[string]string{"albumFrom": "1990", "albumTo": "2000"})
	result := Filters(artists, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists for invalid album year, got %d", len(result))
	}
}

func TestFilters_Members_InvalidValue(t *testing.T) {
	v := url.Values{}
	v.Add("members", "abc")
	r := makeRequestMulti(v)
	result := Filters(testArtists, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists for invalid members param, got %d", len(result))
	}
}

func TestFilters_Location_ArtistNotInLocMap(t *testing.T) {
	// Artist not present in locMap should be filtered out
	artists := []api.Artist{
		{Id: 999, Name: "NoLocations", CreationDate: 2000, Members: []string{"A"}},
	}
	r := makeRequest(map[string]string{"location": "london"})
	result := Filters(artists, testLocMap, r)
	if len(result) != 0 {
		t.Errorf("expected 0 artists when artist not in locMap, got %d", len(result))
	}
}

func TestFilters_CombinedFilters(t *testing.T) {
	// Combine year, album, members, and location filters
	v := url.Values{}
	v.Set("yearFrom", "1990")
	v.Set("yearTo", "2000")
	v.Set("albumFrom", "1995")
	v.Set("albumTo", "2000")
	v.Add("members", "3")
	v.Set("location", "london")
	r := makeRequestMulti(v)
	result := Filters(testArtists, testLocMap, r)
	// Muse: CreationDate=1994, FirstAlbum=1999, Members=3, has london-uk
	if len(result) != 1 || result[0].Name != "Muse" {
		t.Errorf("expected only Muse for combined filters, got %+v", result)
	}
}
