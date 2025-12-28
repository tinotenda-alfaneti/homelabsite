package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
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

	// Create multipart form data
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("id", "new-post")
	writer.WriteField("title", "New Post")
	writer.WriteField("category", "Test")
	writer.WriteField("summary", "Summary")
	writer.WriteField("content", "Content")
	writer.WriteField("tags", `["new"]`)
	writer.WriteField("date", time.Now().Format(time.RFC3339))

	writer.Close()

	req := httptest.NewRequest("POST", "/api/posts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

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

func TestHandleAPISavePostWithFileUpload(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	// Create a temporary test file
	testFileContent := "fake image content"
	testFile, err := os.CreateTemp("", "test_image_*.jpg")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(testFile.Name())

	if _, err := testFile.WriteString(testFileContent); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	testFile.Close()

	// Reopen for reading
	testFile, err = os.Open(testFile.Name())
	if err != nil {
		t.Fatalf("Failed to reopen temp file: %v", err)
	}
	defer testFile.Close()

	// Create multipart form data with file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("id", "post-with-image")
	writer.WriteField("title", "Post with Image")
	writer.WriteField("category", "Test")
	writer.WriteField("summary", "Summary")
	writer.WriteField("content", "Some content")
	writer.WriteField("tags", `["image"]`)
	writer.WriteField("date", time.Now().Format(time.RFC3339))

	// Add file
	part, err := writer.CreatePart(map[string][]string{
		"Content-Disposition": []string{`form-data; name="image"; filename="test_image.jpg"`},
		"Content-Type":        []string{"image/jpeg"},
	})
	if err != nil {
		t.Fatalf("Failed to create form part: %v", err)
	}
	part.Write([]byte("fake image content"))

	writer.Close()

	req := httptest.NewRequest("POST", "/api/posts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPISavePost)
	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusOK)
		t.Errorf("Response body: %s", rr.Body.String())
		return
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Errorf("Failed to decode response: %v", err)
		t.Errorf("Response body: %s", rr.Body.String())
		return
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success to be true")
	}

	// Verify post was saved
	savedPost, err := app.DB.GetPostByID("post-with-image")
	if err != nil {
		t.Errorf("Error getting post: %v", err)
	}
	if savedPost == nil {
		t.Error("Post was not saved to database")
		return
	}

	// Check that image markdown was added to content
	if !strings.Contains(savedPost.Content, "![test_image.jpg](/uploads/") {
		t.Errorf("Expected image markdown in content, got: %s", savedPost.Content)
	}

	// Check that file was saved
	uploadsDir := "web/static/uploads"
	files, err := filepath.Glob(filepath.Join(uploadsDir, "post-with-image_*.jpg"))
	if err != nil {
		t.Errorf("Error checking for uploaded file: %v", err)
	}
	if len(files) == 0 {
		t.Error("Uploaded file was not saved")
	} else {
		// Clean up the uploaded file
		for _, f := range files {
			os.Remove(f)
		}
	}
}

func TestHandleAPISavePostWithInvalidFileType(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	// Create a temporary test file with invalid type
	testFile, err := os.CreateTemp("", "test_script_*.exe")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(testFile.Name())

	if _, err := testFile.WriteString("fake exe content"); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	testFile.Close()

	// Create multipart form data with invalid file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("id", "post-invalid-file")
	writer.WriteField("title", "Post with Invalid File")
	writer.WriteField("category", "Test")
	writer.WriteField("summary", "Summary")
	writer.WriteField("content", "Some content")
	writer.WriteField("tags", `["invalid"]`)
	writer.WriteField("date", time.Now().Format(time.RFC3339))

	// Add file with invalid content type
	fileWriter, err := writer.CreateFormFile("image", "test_script.exe")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	testFile, err = os.Open(testFile.Name())
	if err != nil {
		t.Fatalf("Failed to open temp file: %v", err)
	}
	defer testFile.Close()
	io.Copy(fileWriter, testFile)

	writer.Close()

	req := httptest.NewRequest("POST", "/api/posts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPISavePost)
	handler.ServeHTTP(rr, req)

	// Should return bad request for invalid file type
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
	}
}

func TestHandleAPISavePostWithLargeFile(t *testing.T) {
	app := setupTestApp(t)
	defer teardownTestApp(app)

	// Create multipart form data with large file
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	writer.WriteField("id", "post-large-file")
	writer.WriteField("title", "Post with Large File")
	writer.WriteField("category", "Test")
	writer.WriteField("summary", "Summary")
	writer.WriteField("content", "Some content")
	writer.WriteField("tags", `["large"]`)
	writer.WriteField("date", time.Now().Format(time.RFC3339))

	// Add large file (11MB, exceeds 10MB limit)
	fileWriter, err := writer.CreateFormFile("image", "large_image.jpg")
	if err != nil {
		t.Fatalf("Failed to create form file: %v", err)
	}
	// Write 11MB of data
	largeData := make([]byte, 11<<20) // 11MB
	fileWriter.Write(largeData)

	writer.Close()

	req := httptest.NewRequest("POST", "/api/posts", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(app.HandleAPISavePost)
	handler.ServeHTTP(rr, req)

	// Should return bad request for file too large
	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("Handler returned wrong status code: got %v want %v", status, http.StatusBadRequest)
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
