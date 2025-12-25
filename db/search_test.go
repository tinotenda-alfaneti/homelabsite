package db

import (
	"os"
	"testing"
	"time"

	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func TestSearchPosts(t *testing.T) {
	dbPath := "test_search.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create test posts
	posts := []*models.Post{
		{
			ID:       "kubernetes-basics",
			Title:    "Kubernetes Basics for Beginners",
			Date:     time.Now(),
			Category: "kubernetes",
			Summary:  "Learn Kubernetes fundamentals",
			Content:  "This post covers Kubernetes deployment, pods, and services in detail.",
			Tags:     []string{"kubernetes", "docker", "devops"},
		},
		{
			ID:       "golang-tips",
			Title:    "Advanced Go Programming Tips",
			Date:     time.Now().Add(-1 * time.Hour),
			Category: "golang",
			Summary:  "Pro tips for Go developers",
			Content:  "Explore advanced Go patterns including goroutines and channels.",
			Tags:     []string{"golang", "programming"},
		},
		{
			ID:       "docker-compose",
			Title:    "Docker Compose Tutorial",
			Date:     time.Now().Add(-2 * time.Hour),
			Category: "docker",
			Summary:  "Master docker-compose",
			Content:  "Learn how to use docker-compose for multi-container applications.",
			Tags:     []string{"docker", "devops"},
		},
	}

	for _, post := range posts {
		if err := db.SavePost(post); err != nil {
			t.Fatalf("Failed to save post: %v", err)
		}
	}

	tests := []struct {
		name          string
		query         string
		expectedCount int
		shouldContain string
	}{
		{
			name:          "Search by title",
			query:         "Kubernetes",
			expectedCount: 1,
			shouldContain: "kubernetes-basics",
		},
		{
			name:          "Search by content",
			query:         "goroutines",
			expectedCount: 1,
			shouldContain: "golang-tips",
		},
		{
			name:          "Search by category",
			query:         "golang",
			expectedCount: 1,
			shouldContain: "golang-tips",
		},
		{
			name:          "Search by tag (as text)",
			query:         "devops",
			expectedCount: 2,
			shouldContain: "kubernetes-basics",
		},
		{
			name:          "Search multiple matches",
			query:         "docker",
			expectedCount: 2,
			shouldContain: "docker-compose",
		},
		{
			name:          "Case insensitive search",
			query:         "GOLANG",
			expectedCount: 1,
			shouldContain: "golang-tips",
		},
		{
			name:          "Partial word search",
			query:         "Kuber",
			expectedCount: 1,
			shouldContain: "kubernetes-basics",
		},
		{
			name:          "Empty query",
			query:         "",
			expectedCount: 0,
			shouldContain: "",
		},
		{
			name:          "No matches",
			query:         "python",
			expectedCount: 0,
			shouldContain: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.SearchPosts(tt.query)
			if err != nil {
				t.Fatalf("Search failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			if tt.shouldContain != "" {
				found := false
				for _, post := range results {
					if post.ID == tt.shouldContain {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected to find post %s in results", tt.shouldContain)
				}
			}
		})
	}
}

func TestSearchPostsByTag(t *testing.T) {
	dbPath := "test_tag_search.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create test posts with various tag combinations
	posts := []*models.Post{
		{
			ID:       "post1",
			Title:    "Post 1",
			Date:     time.Now(),
			Category: "test",
			Summary:  "Test",
			Content:  "Content",
			Tags:     []string{"kubernetes", "docker"},
		},
		{
			ID:       "post2",
			Title:    "Post 2",
			Date:     time.Now(),
			Category: "test",
			Summary:  "Test",
			Content:  "Content",
			Tags:     []string{"golang", "kubernetes"},
		},
		{
			ID:       "post3",
			Title:    "Post 3",
			Date:     time.Now(),
			Category: "test",
			Summary:  "Test",
			Content:  "Content",
			Tags:     []string{"docker"},
		},
		{
			ID:       "post4",
			Title:    "Post 4",
			Date:     time.Now(),
			Category: "test",
			Summary:  "Test",
			Content:  "Content",
			Tags:     []string{"python", "django"},
		},
	}

	for _, post := range posts {
		if err := db.SavePost(post); err != nil {
			t.Fatalf("Failed to save post: %v", err)
		}
	}

	tests := []struct {
		name          string
		tag           string
		expectedCount int
		expectedIDs   []string
	}{
		{
			name:          "Search kubernetes tag",
			tag:           "kubernetes",
			expectedCount: 2,
			expectedIDs:   []string{"post1", "post2"},
		},
		{
			name:          "Search docker tag",
			tag:           "docker",
			expectedCount: 2,
			expectedIDs:   []string{"post1", "post3"},
		},
		{
			name:          "Search golang tag",
			tag:           "golang",
			expectedCount: 1,
			expectedIDs:   []string{"post2"},
		},
		{
			name:          "Search non-existent tag",
			tag:           "rust",
			expectedCount: 0,
			expectedIDs:   []string{},
		},
		{
			name:          "Empty tag",
			tag:           "",
			expectedCount: 0,
			expectedIDs:   []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := db.SearchPostsByTag(tt.tag)
			if err != nil {
				t.Fatalf("Tag search failed: %v", err)
			}

			if len(results) != tt.expectedCount {
				t.Errorf("Expected %d results, got %d", tt.expectedCount, len(results))
			}

			// Check that all expected IDs are present
			foundIDs := make(map[string]bool)
			for _, post := range results {
				foundIDs[post.ID] = true
			}

			for _, expectedID := range tt.expectedIDs {
				if !foundIDs[expectedID] {
					t.Errorf("Expected to find post %s in results", expectedID)
				}
			}
		})
	}
}

func TestGetAllTags(t *testing.T) {
	dbPath := "test_all_tags.db"
	defer os.Remove(dbPath)

	db, err := New(dbPath)
	if err != nil {
		t.Fatalf("Failed to create database: %v", err)
	}
	defer db.Close()

	// Create posts with various tags
	posts := []*models.Post{
		{
			ID:       "post1",
			Title:    "Post 1",
			Date:     time.Now(),
			Category: "test",
			Summary:  "Test",
			Content:  "Content",
			Tags:     []string{"kubernetes", "docker", "devops"},
		},
		{
			ID:       "post2",
			Title:    "Post 2",
			Date:     time.Now(),
			Category: "test",
			Summary:  "Test",
			Content:  "Content",
			Tags:     []string{"golang", "kubernetes"},
		},
		{
			ID:       "post3",
			Title:    "Post 3",
			Date:     time.Now(),
			Category: "test",
			Summary:  "Test",
			Content:  "Content",
			Tags:     []string{"python"},
		},
	}

	for _, post := range posts {
		if err := db.SavePost(post); err != nil {
			t.Fatalf("Failed to save post: %v", err)
		}
	}

	tags, err := db.GetAllTags()
	if err != nil {
		t.Fatalf("Failed to get all tags: %v", err)
	}

	// Should have 5 unique tags
	expectedTags := map[string]bool{
		"kubernetes": true,
		"docker":     true,
		"devops":     true,
		"golang":     true,
		"python":     true,
	}

	if len(tags) != len(expectedTags) {
		t.Errorf("Expected %d unique tags, got %d", len(expectedTags), len(tags))
	}

	for _, tag := range tags {
		if !expectedTags[tag] {
			t.Errorf("Unexpected tag: %s", tag)
		}
	}
}
