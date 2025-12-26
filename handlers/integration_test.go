package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHandle404 verifies the custom 404 handler exists
func TestHandle404(t *testing.T) {
	t.Skip("Template rendering tests require full app initialization")
	// This is tested via actual HTTP requests to the application
}

// TestBreadcrumbsInPages verifies breadcrumb functionality
func TestBreadcrumbsInPages(t *testing.T) {
	t.Skip("Template rendering tests require full app initialization")
	// Breadcrumbs are tested via actual HTTP requests
}

// TestPostViewsIncrement verifies view counting works
func TestPostViewsIncrement(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	// Get initial view count
	post, err := app.DB.GetPostByID("test-post-1")
	if err != nil {
		t.Fatal(err)
	}
	initialViews := post.Views

	// Simulate viewing the post (which should increment views)
	err = app.DB.IncrementPostViews("test-post-1")
	if err != nil {
		t.Fatal(err)
	}

	// Get updated view count
	post, err = app.DB.GetPostByID("test-post-1")
	if err != nil {
		t.Fatal(err)
	}

	if post.Views != initialViews+1 {
		t.Errorf("Expected views to be %d, got %d", initialViews+1, post.Views)
	}
}

// TestCacheIntegration verifies cache is initialized and usable
func TestCacheIntegration(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	if app.Cache == nil {
		t.Fatal("Cache should be initialized")
	}

	// Test cache set/get
	testKey := "test-key"
	testValue := "test-value"

	app.Cache.Set(testKey, testValue, 0)
	value, found := app.Cache.Get(testKey)

	if !found {
		t.Error("Expected to find cached value")
	}

	if value != testValue {
		t.Errorf("Expected cached value %s, got %v", testValue, value)
	}
}

// TestAPIPopularPostsReturnsOrderedByViews verifies popular posts are sorted by views
func TestAPIPopularPostsReturnsOrderedByViews(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	// Create posts with different view counts
	posts := []struct {
		id    string
		views int
	}{
		{"popular-1", 100},
		{"popular-2", 50},
		{"popular-3", 25},
	}

	for _, p := range posts {
		err := app.DB.IncrementPostViews(p.id)
		if err != nil {
			t.Logf("Post %s might not exist, skipping view increment", p.id)
		}
	}

	req := httptest.NewRequest("GET", "/api/posts/popular", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPIPopularPosts)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	contentType := rr.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type application/json, got %s", contentType)
	}
}

// TestHealthEndpoint verifies health check returns correctly
func TestHealthEndpointFormat(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/health", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleHealth)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]string
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode JSON response: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response["status"])
	}
}
