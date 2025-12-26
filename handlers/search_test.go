package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestHandleSearch(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	tests := []struct {
		name          string
		query         string
		expectedCount int
		allowEmpty    bool
	}{
		{"Search by title", "Test Post", 1, false},
		{"Search by tag", "test", 1, false},
		{"Search by category", "Testing", 1, false},
		{"Search no results", "nonexistent", 0, true},
		{"Empty query", "", 1, false}, // Returns all posts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/search?q="+url.QueryEscape(tt.query), nil)

			rr := httptest.NewRecorder()
			handler := http.HandlerFunc(app.HandleSearch)
			handler.ServeHTTP(rr, req)

			if status := rr.Code; status != http.StatusOK {
				t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
			}

			var response map[string]interface{}
			if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
				t.Errorf("Failed to decode response: %v", err)
			}

			posts, ok := response["posts"].([]interface{})
			if !ok {
				// If allowEmpty, check if it's nil
				if tt.allowEmpty && response["posts"] == nil {
					return // This is acceptable for "no results"
				}
				t.Errorf("Expected posts array in response")
				return
			}

			if len(posts) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(posts))
			}
		})
	}
}

func TestHandleAPITags(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/api/tags", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPITags)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	tagsInterface, ok := response["tags"].([]interface{})
	if !ok {
		t.Error("Expected tags array in response")
		return
	}

	if len(tagsInterface) == 0 {
		t.Error("Expected at least one tag")
	}

	// Verify tags include expected values
	foundTestTag := false
	for _, tagInterface := range tagsInterface {
		tag, ok := tagInterface.(string)
		if !ok {
			continue
		}
		if tag == "test" || tag == "go" {
			foundTestTag = true
			break
		}
	}

	if !foundTestTag {
		t.Error("Expected to find 'test' or 'go' tag")
	}
}
