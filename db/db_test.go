package db

import (
	"os"
	"testing"
	"time"

	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func TestNew(t *testing.T) {
	dbPath := "test_homelab.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Verify connection works
	if err := db.conn.Ping(); err != nil {
		t.Errorf("Database ping failed: %v", err)
	}
}

func TestSaveAndGetPost(t *testing.T) {
	dbPath := "test_posts.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create test post
	post := &models.Post{
		ID:       "test-post",
		Title:    "Test Post",
		Date:     time.Now(),
		Category: "testing",
		Summary:  "A test post",
		Content:  "This is test content",
		Tags:     []string{"test", "demo"},
	}

	// Save post
	if err := db.SavePost(post); err != nil {
		t.Fatalf("Failed to save post: %v", err)
	}

	// Retrieve post
	retrieved, err := db.GetPostByID("test-post")
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected post to be found")
	}

	if retrieved.Title != post.Title {
		t.Errorf("Expected title '%s', got '%s'", post.Title, retrieved.Title)
	}

	if retrieved.Category != post.Category {
		t.Errorf("Expected category '%s', got '%s'", post.Category, retrieved.Category)
	}

	if len(retrieved.Tags) != len(post.Tags) {
		t.Errorf("Expected %d tags, got %d", len(post.Tags), len(retrieved.Tags))
	}
}

func TestUpdatePost(t *testing.T) {
	dbPath := "test_update.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create and save initial post
	post := &models.Post{
		ID:       "update-test",
		Title:    "Original Title",
		Date:     time.Now(),
		Category: "original",
		Summary:  "Original summary",
		Content:  "Original content",
		Tags:     []string{"original"},
	}
	db.SavePost(post)

	// Update post
	post.Title = "Updated Title"
	post.Category = "updated"
	if err := db.SavePost(post); err != nil {
		t.Fatalf("Failed to update post: %v", err)
	}

	// Retrieve updated post
	retrieved, err := db.GetPostByID("update-test")
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected title 'Updated Title', got '%s'", retrieved.Title)
	}
}

func TestDeletePost(t *testing.T) {
	dbPath := "test_delete.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create and save post
	post := &models.Post{
		ID:       "delete-test",
		Title:    "To Be Deleted",
		Date:     time.Now(),
		Category: "test",
		Summary:  "Test",
		Content:  "Test",
		Tags:     []string{},
	}
	db.SavePost(post)

	// Delete post
	if err := db.DeletePost("delete-test"); err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}

	// Try to retrieve deleted post
	retrieved, err := db.GetPostByID("delete-test")
	if err != nil {
		t.Fatalf("Error checking for deleted post: %v", err)
	}

	if retrieved != nil {
		t.Error("Expected post to be deleted, but it was found")
	}
}

func TestGetAllPosts(t *testing.T) {
	dbPath := "test_getall.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create multiple posts
	posts := []*models.Post{
		{
			ID:       "post1",
			Title:    "Post 1",
			Date:     time.Now().Add(-2 * time.Hour),
			Category: "test",
			Summary:  "Test 1",
			Content:  "Content 1",
			Tags:     []string{},
		},
		{
			ID:       "post2",
			Title:    "Post 2",
			Date:     time.Now().Add(-1 * time.Hour),
			Category: "test",
			Summary:  "Test 2",
			Content:  "Content 2",
			Tags:     []string{},
		},
	}

	for _, post := range posts {
		if err := db.SavePost(post); err != nil {
			t.Fatalf("Failed to save post: %v", err)
		}
	}

	// Get all posts
	retrieved, err := db.GetAllPosts()
	if err != nil {
		t.Fatalf("Failed to get all posts: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Expected 2 posts, got %d", len(retrieved))
	}

	// Posts should be ordered by date descending
	if len(retrieved) >= 2 && retrieved[0].Date.Before(retrieved[1].Date) {
		t.Error("Expected posts to be ordered by date descending")
	}
}

func TestSaveAndGetService(t *testing.T) {
	dbPath := "test_service.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create test service
	service := &models.Service{
		Name:        "Test Service",
		Description: "A test service",
		URL:         "https://test.example.com",
		Tech:        "Go",
		Status:      "active",
		Icon:        "🧪",
	}

	// Save service
	if err := db.SaveService(service); err != nil {
		t.Fatalf("Failed to save service: %v", err)
	}

	// Get all services
	services, err := db.GetAllServices()
	if err != nil {
		t.Fatalf("Failed to get services: %v", err)
	}

	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}

	if services[0].Name != service.Name {
		t.Errorf("Expected name '%s', got '%s'", service.Name, services[0].Name)
	}
}
