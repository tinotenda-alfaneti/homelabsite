package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/cache"
	"github.com/tinotenda-alfaneti/homelabsite/db"
	"github.com/tinotenda-alfaneti/homelabsite/middleware"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func setupTestApp(t *testing.T) *App {
	// Create temporary test database
	dbPath := "/tmp/test_homelab.db"
	os.Remove(dbPath) // Clean up any existing test db

	database, err := db.New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	// Seed test data
	testPost := &models.Post{
		ID:       "test-post-1",
		Title:    "Test Post 1",
		Date:     time.Now(),
		Category: "Testing",
		Summary:  "A test post summary",
		Content:  "Test post content",
		Tags:     []string{"test", "go"},
		Views:    10,
	}
	if err := database.SavePost(testPost); err != nil {
		t.Fatalf("Failed to save test post: %v", err)
	}

	testService := &models.Service{
		Name:        "Test Service",
		Description: "A test service",
		URL:         "https://test.example.com",
		Tech:        "Go",
		Status:      "active",
		Icon:        "🧪",
	}
	if err := database.SaveService(testService); err != nil {
		t.Fatalf("Failed to save test service: %v", err)
	}

	return &App{
		DB:    database,
		Auth:  middleware.NewAuthMiddleware("admin", "password"),
		Cache: cache.New(),
	}
}

func teardownTestApp(app *App) {
	if app.DB != nil {
		app.DB.Close()
	}
	os.Remove("/tmp/test_homelab.db")
}

func TestHandleAPIServices(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/api/services", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPIServices)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var services []models.Service
	if err := json.NewDecoder(rr.Body).Decode(&services); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(services) == 0 {
		t.Error("Expected at least one service")
	}

	if services[0].Name != "Test Service" {
		t.Errorf("Expected service name 'Test Service', got '%s'", services[0].Name)
	}
}

func TestHandleAPIServicesWithStatusFilter(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/api/services?status=active", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPIServices)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var services []models.Service
	if err := json.NewDecoder(rr.Body).Decode(&services); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	for _, s := range services {
		if s.Status != "active" {
			t.Errorf("Expected only active services, got status: %s", s.Status)
		}
	}
}

func TestHandleAPIPosts(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/api/posts", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPIPosts)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var posts []models.Post
	if err := json.NewDecoder(rr.Body).Decode(&posts); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(posts) == 0 {
		t.Error("Expected at least one post")
	}
}

func TestHandleAPIGetPost(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/api/posts/test-post-1", nil)

	rr := httptest.NewRecorder()

	// Use mux router to handle path variables
	r := mux.NewRouter()
	r.HandleFunc("/api/posts/{id}", app.HandleAPIGetPost)
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var post models.Post
	if err := json.NewDecoder(rr.Body).Decode(&post); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if post.ID != "test-post-1" {
		t.Errorf("Expected post ID 'test-post-1', got '%s'", post.ID)
	}

	if post.Views != 10 {
		t.Errorf("Expected post views 10, got %d", post.Views)
	}
}

func TestHandleAPIGetPostNotFound(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/api/posts/nonexistent", nil)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/api/posts/{id}", app.HandleAPIGetPost)
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusNotFound {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusNotFound)
	}
}

func TestHandleAPISavePost(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	newPost := models.Post{
		ID:       "new-post",
		Title:    "New Post",
		Date:     time.Now(),
		Category: "Test",
		Summary:  "Summary",
		Content:  "Content",
		Tags:     []string{"new"},
	}

	body, _ := json.Marshal(newPost)
	req := httptest.NewRequest("POST", "/api/posts", bytes.NewBuffer(body))

	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPISavePost)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}

	// Verify post was saved
	savedPost, _ := app.DB.GetPostByID("new-post")
	if savedPost == nil {
		t.Error("Post was not saved to database")
	}
}

func TestHandleAPIDeletePost(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("DELETE", "/api/posts/test-post-1", nil)

	rr := httptest.NewRecorder()

	r := mux.NewRouter()
	r.HandleFunc("/api/posts/{id}", app.HandleAPIDeletePost)
	r.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}

	// Verify post was deleted
	deletedPost, _ := app.DB.GetPostByID("test-post-1")
	if deletedPost != nil {
		t.Error("Post was not deleted from database")
	}
}

func TestHandleAPIPopularPosts(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	// Add more posts with different view counts
	posts := []*models.Post{
		{
			ID:       "popular-1",
			Title:    "Popular Post 1",
			Date:     time.Now(),
			Category: "Test",
			Summary:  "Summary",
			Content:  "Content",
			Tags:     []string{"popular"},
			Views:    100,
		},
		{
			ID:       "popular-2",
			Title:    "Popular Post 2",
			Date:     time.Now(),
			Category: "Test",
			Summary:  "Summary",
			Content:  "Content",
			Tags:     []string{"popular"},
			Views:    50,
		},
	}

	for _, p := range posts {
		if err := app.DB.SavePost(p); err != nil {
			t.Fatalf("Failed to save post: %v", err)
		}
	}

	req := httptest.NewRequest("GET", "/api/posts/popular", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPIPopularPosts)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var popularPosts []models.Post
	if err := json.NewDecoder(rr.Body).Decode(&popularPosts); err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(popularPosts) == 0 {
		t.Error("Expected at least one popular post")
	}

	// Verify posts are ordered by views descending
	for i := 0; i < len(popularPosts)-1; i++ {
		if popularPosts[i].Views < popularPosts[i+1].Views {
			t.Error("Posts are not ordered by views descending")
		}
	}
}

func TestHandleHealth(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	req := httptest.NewRequest("GET", "/health", nil)

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleHealth)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"status":"healthy"}`
	if rr.Body.String() != expected {
		t.Errorf("Handler returned unexpected body: got %v want %v", rr.Body.String(), expected)
	}
}
