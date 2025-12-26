package db

import (
	"os"
	"testing"
	"time"

	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func setupTestDB(t *testing.T) (*DB, func()) {
	dbPath := "/tmp/test_views.db"
	os.Remove(dbPath) // Clean up

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.Remove(dbPath)
	}

	return db, cleanup
}

func TestIncrementPostViews(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a test post
	post := &models.Post{
		ID:       "test-views-1",
		Title:    "Test Post",
		Date:     time.Now(),
		Category: "Test",
		Summary:  "Test summary",
		Content:  "Test content",
		Tags:     []string{"test"},
		Views:    0,
	}

	err := db.SavePost(post)
	if err != nil {
		t.Fatalf("Failed to save post: %v", err)
	}

	// Increment views
	err = db.IncrementPostViews("test-views-1")
	if err != nil {
		t.Fatalf("Failed to increment views: %v", err)
	}

	// Retrieve post and check views
	retrievedPost, err := db.GetPostByID("test-views-1")
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	if retrievedPost.Views != 1 {
		t.Errorf("Expected views to be 1, got %d", retrievedPost.Views)
	}

	// Increment again
	err = db.IncrementPostViews("test-views-1")
	if err != nil {
		t.Fatalf("Failed to increment views second time: %v", err)
	}

	// Check views again
	retrievedPost, err = db.GetPostByID("test-views-1")
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	if retrievedPost.Views != 2 {
		t.Errorf("Expected views to be 2, got %d", retrievedPost.Views)
	}
}

func TestGetPopularPosts(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create multiple posts with different view counts
	posts := []*models.Post{
		{
			ID:       "popular-1",
			Title:    "Most Popular",
			Date:     time.Now(),
			Category: "Test",
			Summary:  "Summary 1",
			Content:  "Content 1",
			Tags:     []string{"test"},
			Views:    100,
		},
		{
			ID:       "popular-2",
			Title:    "Second Popular",
			Date:     time.Now(),
			Category: "Test",
			Summary:  "Summary 2",
			Content:  "Content 2",
			Tags:     []string{"test"},
			Views:    50,
		},
		{
			ID:       "popular-3",
			Title:    "Third Popular",
			Date:     time.Now(),
			Category: "Test",
			Summary:  "Summary 3",
			Content:  "Content 3",
			Tags:     []string{"test"},
			Views:    25,
		},
		{
			ID:       "unpopular",
			Title:    "Unpopular",
			Date:     time.Now(),
			Category: "Test",
			Summary:  "Summary 4",
			Content:  "Content 4",
			Tags:     []string{"test"},
			Views:    5,
		},
	}

	for _, post := range posts {
		err := db.SavePost(post)
		if err != nil {
			t.Fatalf("Failed to save post %s: %v", post.ID, err)
		}
	}

	// Get top 3 popular posts
	popularPosts, err := db.GetPopularPosts(3)
	if err != nil {
		t.Fatalf("Failed to get popular posts: %v", err)
	}

	if len(popularPosts) != 3 {
		t.Errorf("Expected 3 popular posts, got %d", len(popularPosts))
	}

	// Check ordering by views (descending)
	if popularPosts[0].ID != "popular-1" || popularPosts[0].Views != 100 {
		t.Errorf("Expected first post to be popular-1 with 100 views")
	}

	if popularPosts[1].ID != "popular-2" || popularPosts[1].Views != 50 {
		t.Errorf("Expected second post to be popular-2 with 50 views")
	}

	if popularPosts[2].ID != "popular-3" || popularPosts[2].Views != 25 {
		t.Errorf("Expected third post to be popular-3 with 25 views")
	}
}

func TestIncrementNonexistentPost(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Try to increment views on non-existent post (should not error)
	err := db.IncrementPostViews("nonexistent-post")
	if err != nil {
		t.Errorf("Incrementing nonexistent post should not error: %v", err)
	}
}

func TestViewsPersistence(t *testing.T) {
	dbPath := "/tmp/test_views_persist.db"
	os.Remove(dbPath)

	// Create database and save post with views
	db1, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}

	post := &models.Post{
		ID:       "persist-test",
		Title:    "Persistence Test",
		Date:     time.Now(),
		Category: "Test",
		Summary:  "Test summary",
		Content:  "Test content",
		Tags:     []string{"test"},
		Views:    42,
	}

	err = db1.SavePost(post)
	if err != nil {
		t.Fatalf("Failed to save post: %v", err)
	}

	db1.Close()

	// Reopen database and check views persisted
	db2, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db2.Close()
	defer os.Remove(dbPath)

	retrievedPost, err := db2.GetPostByID("persist-test")
	if err != nil {
		t.Fatalf("Failed to get post: %v", err)
	}

	if retrievedPost.Views != 42 {
		t.Errorf("Expected views to persist as 42, got %d", retrievedPost.Views)
	}
}
