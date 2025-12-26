package db

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func setupCommentsTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Enable foreign key constraints in SQLite
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatalf("Failed to enable foreign keys: %v", err)
	}

	// Create posts table
	postsSchema := `
	CREATE TABLE IF NOT EXISTS posts (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		date DATETIME NOT NULL,
		category TEXT NOT NULL,
		summary TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT NOT NULL
	);`

	if _, err := db.Exec(postsSchema); err != nil {
		t.Fatalf("Failed to create posts table: %v", err)
	}

	// Create comments table
	if err := CreateCommentsTable(db); err != nil {
		t.Fatalf("Failed to create comments table: %v", err)
	}

	// Insert test post
	_, err = db.Exec(`
		INSERT INTO posts (id, title, date, category, summary, content, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, "test-post", "Test Post", time.Now(), "Test", "A test post", "Test content", "test")

	if err != nil {
		t.Fatalf("Failed to insert test post: %v", err)
	}

	return db
}

func TestSaveComment(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	comment := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "John Doe",
		AuthorEmail: "john@example.com",
		Content:     "This is a test comment",
		CreatedAt:   time.Now(),
		Approved:    false,
	}

	err := SaveComment(db, comment)
	if err != nil {
		t.Fatalf("Failed to save comment: %v", err)
	}

	if comment.ID == 0 {
		t.Error("Comment ID should be set after saving")
	}
}

func TestGetCommentsByPostID(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	// Save approved comment
	approvedComment := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "Alice",
		AuthorEmail: "alice@example.com",
		Content:     "Approved comment",
		CreatedAt:   time.Now(),
		Approved:    true,
	}
	if err := SaveComment(db, approvedComment); err != nil {
		t.Fatalf("Failed to save approved comment: %v", err)
	}

	// Save unapproved comment
	unapprovedComment := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "Bob",
		AuthorEmail: "bob@example.com",
		Content:     "Pending comment",
		CreatedAt:   time.Now(),
		Approved:    false,
	}
	if err := SaveComment(db, unapprovedComment); err != nil {
		t.Fatalf("Failed to save unapproved comment: %v", err)
	}

	// Get comments (should only return approved)
	comments, err := GetCommentsByPostID(db, "test-post")
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	if len(comments) != 1 {
		t.Errorf("Expected 1 approved comment, got %d", len(comments))
	}

	if comments[0].AuthorName != "Alice" {
		t.Errorf("Expected Alice's comment, got %s", comments[0].AuthorName)
	}
}

func TestGetPendingComments(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	// Save pending comments
	for i := 0; i < 3; i++ {
		comment := &models.Comment{
			PostID:      "test-post",
			AuthorName:  "User",
			AuthorEmail: "user@example.com",
			Content:     "Pending comment",
			CreatedAt:   time.Now(),
			Approved:    false,
		}
		if err := SaveComment(db, comment); err != nil {
			t.Fatalf("Failed to save comment %d: %v", i, err)
		}
	}

	// Save approved comment
	approved := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "Approved User",
		AuthorEmail: "approved@example.com",
		Content:     "Approved",
		CreatedAt:   time.Now(),
		Approved:    true,
	}
	if err := SaveComment(db, approved); err != nil {
		t.Fatalf("Failed to save approved comment: %v", err)
	}

	pending, err := GetPendingComments(db)
	if err != nil {
		t.Fatalf("Failed to get pending comments: %v", err)
	}

	if len(pending) != 3 {
		t.Errorf("Expected 3 pending comments, got %d", len(pending))
	}
}

func TestApproveComment(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	comment := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "Tester",
		AuthorEmail: "tester@example.com",
		Content:     "Test",
		CreatedAt:   time.Now(),
		Approved:    false,
	}
	if err := SaveComment(db, comment); err != nil {
		t.Fatalf("Failed to save comment: %v", err)
	}

	// Approve the comment
	err := ApproveComment(db, comment.ID)
	if err != nil {
		t.Fatalf("Failed to approve comment: %v", err)
	}

	// Verify it's now approved
	comments, _ := GetCommentsByPostID(db, "test-post")
	if len(comments) != 1 {
		t.Errorf("Expected 1 approved comment, got %d", len(comments))
	}
}

func TestDeleteComment(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	comment := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "Delete Me",
		AuthorEmail: "delete@example.com",
		Content:     "This will be deleted",
		CreatedAt:   time.Now(),
		Approved:    true,
	}
	if err := SaveComment(db, comment); err != nil {
		t.Fatalf("Failed to save comment: %v", err)
	}

	// Delete the comment
	err := DeleteComment(db, comment.ID)
	if err != nil {
		t.Fatalf("Failed to delete comment: %v", err)
	}

	// Verify it's deleted
	comments, _ := GetCommentsByPostID(db, "test-post")
	if len(comments) != 0 {
		t.Errorf("Expected 0 comments after deletion, got %d", len(comments))
	}
}

func TestNestedComments(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	// Parent comment
	parent := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "Parent",
		AuthorEmail: "parent@example.com",
		Content:     "Parent comment",
		CreatedAt:   time.Now(),
		Approved:    true,
	}
	if err := SaveComment(db, parent); err != nil {
		t.Fatalf("Failed to save parent comment: %v", err)
	}

	// Reply to parent
	reply := &models.Comment{
		PostID:      "test-post",
		ParentID:    &parent.ID,
		AuthorName:  "Child",
		AuthorEmail: "child@example.com",
		Content:     "Reply to parent",
		CreatedAt:   time.Now(),
		Approved:    true,
	}
	if err := SaveComment(db, reply); err != nil {
		t.Fatalf("Failed to save reply comment: %v", err)
	}

	// Get all comments
	comments, err := GetCommentsByPostID(db, "test-post")
	if err != nil {
		t.Fatalf("Failed to get comments: %v", err)
	}

	if len(comments) != 2 {
		t.Errorf("Expected 2 comments, got %d", len(comments))
	}

	// Verify reply has parent ID
	var foundReply *models.Comment
	for i := range comments {
		if comments[i].AuthorName == "Child" {
			foundReply = &comments[i]
			break
		}
	}

	if foundReply == nil {
		t.Fatal("Reply comment not found")
	}

	if foundReply.ParentID == nil {
		t.Error("Reply should have a parent ID")
	} else if *foundReply.ParentID != parent.ID {
		t.Errorf("Expected parent ID %d, got %d", parent.ID, *foundReply.ParentID)
	}
}

func TestGetCommentCount(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	// Add 5 approved comments
	for i := 0; i < 5; i++ {
		comment := &models.Comment{
			PostID:      "test-post",
			AuthorName:  "User",
			AuthorEmail: "user@example.com",
			Content:     "Comment",
			CreatedAt:   time.Now(),
			Approved:    true,
		}
		if err := SaveComment(db, comment); err != nil {
			t.Fatalf("Failed to save comment %d: %v", i, err)
		}
	}

	// Add 2 pending comments
	for i := 0; i < 2; i++ {
		comment := &models.Comment{
			PostID:      "test-post",
			AuthorName:  "Pending User",
			AuthorEmail: "pending@example.com",
			Content:     "Pending",
			CreatedAt:   time.Now(),
			Approved:    false,
		}
		if err := SaveComment(db, comment); err != nil {
			t.Fatalf("Failed to save pending comment %d: %v", i, err)
		}
	}

	count, err := GetCommentCount(db, "test-post")
	if err != nil {
		t.Fatalf("Failed to get comment count: %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 approved comments, got %d", count)
	}
}

func TestCascadeDeleteComments(t *testing.T) {
	db := setupCommentsTestDB(t)
	defer db.Close()

	// Add comment
	comment := &models.Comment{
		PostID:      "test-post",
		AuthorName:  "User",
		AuthorEmail: "user@example.com",
		Content:     "Comment on post",
		CreatedAt:   time.Now(),
		Approved:    true,
	}
	if err := SaveComment(db, comment); err != nil {
		t.Fatalf("Failed to save comment: %v", err)
	}

	// Delete the post (should cascade delete comments)
	_, err := db.Exec("DELETE FROM posts WHERE id = ?", "test-post")
	if err != nil {
		t.Fatalf("Failed to delete post: %v", err)
	}

	// Verify comment is also deleted
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = ?", "test-post").Scan(&count)
	if err != nil {
		t.Fatalf("Failed to count comments: %v", err)
	}

	if count != 0 {
		t.Errorf("Expected 0 comments after post deletion, got %d", count)
	}
}
