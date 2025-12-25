package db

import (
	"database/sql"
	"time"

	"github.com/tinotenda-alfaneti/homelab/models"
)

// CreateCommentsTable initializes the comments table in the database
func CreateCommentsTable(database *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS comments (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		post_id TEXT NOT NULL,
		parent_id INTEGER DEFAULT NULL,
		author_name TEXT NOT NULL,
		author_email TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		approved BOOLEAN DEFAULT 0,
		FOREIGN KEY (post_id) REFERENCES posts(id) ON DELETE CASCADE,
		FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE
	);
	
	CREATE INDEX IF NOT EXISTS idx_comments_post_id ON comments(post_id);
	CREATE INDEX IF NOT EXISTS idx_comments_parent_id ON comments(parent_id);
	CREATE INDEX IF NOT EXISTS idx_comments_approved ON comments(approved);
	`
	_, err := database.Exec(query)
	return err
}

// SaveComment inserts a new comment into the database
func SaveComment(database *sql.DB, comment *models.Comment) error {
	tx, err := database.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var parentID interface{}
	if comment.ParentID != nil {
		parentID = *comment.ParentID
	}

	result, err := tx.Exec(`
		INSERT INTO comments (post_id, parent_id, author_name, author_email, content, created_at, approved)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, comment.PostID, parentID, comment.AuthorName, comment.AuthorEmail, comment.Content, comment.CreatedAt, comment.Approved)

	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	comment.ID = int(id)
	return tx.Commit()
}

// GetCommentsByPostID retrieves all approved comments for a specific post
func GetCommentsByPostID(database *sql.DB, postID string) ([]models.Comment, error) {
	rows, err := database.Query(`
		SELECT id, post_id, parent_id, author_name, author_email, content, created_at, approved
		FROM comments
		WHERE post_id = ? AND approved = 1
		ORDER BY created_at ASC
	`, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		var parentID sql.NullInt64
		
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&parentID,
			&comment.AuthorName,
			&comment.AuthorEmail,
			&comment.Content,
			&comment.CreatedAt,
			&comment.Approved,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			comment.ParentID = &pid
		}

		comments = append(comments, comment)
	}

	return comments, rows.Err()
}

// GetPendingComments retrieves all comments pending approval
func GetPendingComments(database *sql.DB) ([]models.Comment, error) {
	rows, err := database.Query(`
		SELECT id, post_id, parent_id, author_name, author_email, content, created_at, approved
		FROM comments
		WHERE approved = 0
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []models.Comment
	for rows.Next() {
		var comment models.Comment
		var parentID sql.NullInt64
		
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&parentID,
			&comment.AuthorName,
			&comment.AuthorEmail,
			&comment.Content,
			&comment.CreatedAt,
			&comment.Approved,
		)
		if err != nil {
			return nil, err
		}

		if parentID.Valid {
			pid := int(parentID.Int64)
			comment.ParentID = &pid
		}

		comments = append(comments, comment)
	}

	return comments, rows.Err()
}

// ApproveComment sets a comment's approved status to true
func ApproveComment(database *sql.DB, commentID int) error {
	_, err := database.Exec(`
		UPDATE comments SET approved = 1 WHERE id = ?
	`, commentID)
	return err
}

// DeleteComment removes a comment from the database
func DeleteComment(database *sql.DB, commentID int) error {
	_, err := database.Exec(`
		DELETE FROM comments WHERE id = ?
	`, commentID)
	return err
}

// GetCommentCount returns the total number of approved comments for a post
func GetCommentCount(database *sql.DB, postID string) (int, error) {
	var count int
	err := database.QueryRow(`
		SELECT COUNT(*) FROM comments WHERE post_id = ? AND approved = 1
	`, postID).Scan(&count)
	return count, err
}
