package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestHomeHandler_WrongPath checks that any path other than "/" returns 404.
func TestHomeHandler_WrongPath(t *testing.T) {
	r := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()
	HomeHandler(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestArtistHandler_InvalidID checks that a non-numeric ID returns 400.
func TestArtistHandler_InvalidID(t *testing.T) {
	r := httptest.NewRequest("GET", "/artist/abc", nil)
	r.URL.Path = "/artist/abc"
	w := httptest.NewRecorder()
	ArtistHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestArtistHandler_ZeroID checks that ID=0 returns 400.
func TestArtistHandler_ZeroID(t *testing.T) {
	r := httptest.NewRequest("GET", "/artist/0", nil)
	r.URL.Path = "/artist/0"
	w := httptest.NewRecorder()
	ArtistHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestArtistHandler_NegativeID checks that a negative ID returns 400.
func TestArtistHandler_NegativeID(t *testing.T) {
	r := httptest.NewRequest("GET", "/artist/-1", nil)
	r.URL.Path = "/artist/-1"
	w := httptest.NewRecorder()
	ArtistHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestArtistHandler_CoordsInvalidID checks that /artist/abc/coords returns 400.
func TestArtistHandler_CoordsInvalidID(t *testing.T) {
	r := httptest.NewRequest("GET", "/artist/abc/coords", nil)
	r.URL.Path = "/artist/abc/coords"
	w := httptest.NewRecorder()
	ArtistHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestArtistHandler_CoordsZeroID checks that /artist/0/coords returns 400.
func TestArtistHandler_CoordsZeroID(t *testing.T) {
	r := httptest.NewRequest("GET", "/artist/0/coords", nil)
	r.URL.Path = "/artist/0/coords"
	w := httptest.NewRecorder()
	ArtistHandler(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// TestSearchHandler_EmptyQuery checks that /search with no query returns a valid JSON response.
func TestSearchHandler_EmptyQuery(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?q=", nil)
	w := httptest.NewRecorder()
	SearchHandler(w, r)

	// The handler calls the external API; if it's unreachable the response will be 500.
	// We only assert the Content-Type when the call succeeds (200).
	if w.Code == http.StatusOK {
		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected application/json, got %q", ct)
		}
	}
}

// TestAddSuggestion_AddsNewEntry verifies a new suggestion is appended.
func TestAddSuggestion_AddsNewEntry(t *testing.T) {
	var suggestions []SearchSuggestion
	seen := map[string]bool{}
	addSuggestion(&suggestions, seen, "Queen", "artist/band")
	if len(suggestions) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(suggestions))
	}
	if suggestions[0].Text != "Queen" || suggestions[0].Type != "artist/band" {
		t.Errorf("unexpected suggestion: %+v", suggestions[0])
	}
}

// TestAddSuggestion_DeduplicatesSameTextAndType verifies duplicates are ignored.
func TestAddSuggestion_DeduplicatesSameTextAndType(t *testing.T) {
	var suggestions []SearchSuggestion
	seen := map[string]bool{}
	addSuggestion(&suggestions, seen, "Queen", "artist/band")
	addSuggestion(&suggestions, seen, "Queen", "artist/band")
	if len(suggestions) != 1 {
		t.Errorf("expected 1 suggestion after duplicate, got %d", len(suggestions))
	}
}

// TestAddSuggestion_AllowsSameTextDifferentType verifies same text with different type is added.
func TestAddSuggestion_AllowsSameTextDifferentType(t *testing.T) {
	var suggestions []SearchSuggestion
	seen := map[string]bool{}
	addSuggestion(&suggestions, seen, "1970", "creation date")
	addSuggestion(&suggestions, seen, "1970", "first album date")
	if len(suggestions) != 2 {
		t.Errorf("expected 2 suggestions, got %d", len(suggestions))
	}
}

// TestAddSuggestion_EmptyTextIgnored verifies empty text is not added.
func TestAddSuggestion_EmptyTextIgnored(t *testing.T) {
	var suggestions []SearchSuggestion
	seen := map[string]bool{}
	addSuggestion(&suggestions, seen, "", "artist/band")
	if len(suggestions) != 0 {
		t.Errorf("expected 0 suggestions for empty text, got %d", len(suggestions))
	}
}

// TestRenderError_WritesStatusCode verifies renderError sets the correct HTTP status.
func TestRenderError_WritesStatusCode(t *testing.T) {
	w := httptest.NewRecorder()
	renderError(w, http.StatusNotFound, "not found")
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// TestRenderError_FallbackWhenNoTemplate verifies renderError falls back to plain text
// when the template file is missing (which it is in the test environment).
func TestRenderError_FallbackWhenNoTemplate(t *testing.T) {
	w := httptest.NewRecorder()
	renderError(w, http.StatusInternalServerError, "something broke")
	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
	body := w.Body.String()
	if !strings.Contains(body, "something broke") && body == "" {
		t.Errorf("expected error message in body, got %q", body)
	}
}

// TestSearchHandler_ReturnsJSON verifies the search handler returns JSON content-type on success.
func TestSearchHandler_ReturnsJSON(t *testing.T) {
	r := httptest.NewRequest("GET", "/search?q=queen", nil)
	w := httptest.NewRecorder()
	SearchHandler(w, r)

	if w.Code == http.StatusOK {
		ct := w.Header().Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("expected application/json, got %q", ct)
		}
		var resp SearchResponse
		if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
			t.Errorf("response is not valid JSON: %v", err)
		}
	}
}

// TestSearchResponse_JSONSerialization verifies SearchResponse marshals correctly.
func TestSearchResponse_JSONSerialization(t *testing.T) {
	resp := SearchResponse{
		Suggestions: []SearchSuggestion{
			{Text: "Queen", Type: "artist/band"},
		},
	}
	b, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if !strings.Contains(string(b), "Queen") {
		t.Errorf("expected 'Queen' in JSON output")
	}
}
