package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/db"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

// HandleGetComments returns all approved comments for a post
func HandleGetComments(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		postID := vars["id"]

		comments, err := db.GetCommentsByPostID(database.GetConn(), postID)
		if err != nil {
			http.Error(w, "Failed to retrieve comments", http.StatusInternalServerError)
			log.Printf("Error getting comments for post %s: %v", postID, err)
			return
		}

		// Build comment tree (nest replies under parents)
		commentTree := buildCommentTree(comments)

		// Check if HTMX request (return HTML) or regular API request (return JSON)
		if r.Header.Get("HX-Request") == "true" {
			w.Header().Set("Content-Type", "text/html")
			renderCommentsHTML(w, commentTree)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"comments": commentTree,
			})
		}
	}
}

// HandlePostComment creates a new comment (requires moderation)
func HandlePostComment(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		postID := vars["id"]

		// Parse form data
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form data", http.StatusBadRequest)
			return
		}

		authorName := strings.TrimSpace(r.FormValue("author_name"))
		authorEmail := strings.TrimSpace(r.FormValue("author_email"))
		content := strings.TrimSpace(r.FormValue("content"))

		// Validation
		if authorName == "" {
			renderFormError(w, "Author name is required")
			return
		}
		if authorEmail == "" {
			renderFormError(w, "Author email is required")
			return
		}
		if content == "" {
			renderFormError(w, "Comment content is required")
			return
		}
		if len(content) > 2000 {
			renderFormError(w, "Comment content too long (max 2000 chars)")
			return
		}

		comment := &models.Comment{
			PostID:      postID,
			ParentID:    nil,
			AuthorName:  authorName,
			AuthorEmail: authorEmail,
			Content:     content,
			CreatedAt:   time.Now(),
			Approved:    false, // Requires admin approval
		}

		if err := db.SaveComment(database.GetConn(), comment); err != nil {
			http.Error(w, "Failed to save comment", http.StatusInternalServerError)
			log.Printf("Error saving comment: %v", err)
			return
		}

		// Return success message for HTMX
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<div style="color: green; font-weight: bold;">✓ Comment submitted for moderation. It will appear after approval.</div>`)
	}
}

// renderFormError renders an error message for the comment form
func renderFormError(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusBadRequest)
	fmt.Fprintf(w, `<div style="color: red; font-weight: bold;">✗ %s</div>`, message)
}

// HandleGetPendingComments returns all comments pending approval (admin only)
func HandleGetPendingComments(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		comments, err := db.GetPendingComments(database.GetConn())
		if err != nil {
			http.Error(w, "Failed to retrieve pending comments", http.StatusInternalServerError)
			log.Printf("Error getting pending comments: %v", err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"comments": comments,
		})
	}
}

// HandleApproveComment approves a comment (admin only)
func HandleApproveComment(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		commentIDStr := vars["id"]
		
		commentID, err := strconv.Atoi(commentIDStr)
		if err != nil {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}

		if err := db.ApproveComment(database.GetConn(), commentID); err != nil {
			http.Error(w, "Failed to approve comment", http.StatusInternalServerError)
			log.Printf("Error approving comment %d: %v", commentID, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Comment approved",
		})
	}
}

// HandleDeleteComment deletes a comment (admin only)
func HandleDeleteComment(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		commentIDStr := vars["id"]
		
		commentID, err := strconv.Atoi(commentIDStr)
		if err != nil {
			http.Error(w, "Invalid comment ID", http.StatusBadRequest)
			return
		}

		if err := db.DeleteComment(database.GetConn(), commentID); err != nil {
			http.Error(w, "Failed to delete comment", http.StatusInternalServerError)
			log.Printf("Error deleting comment %d: %v", commentID, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Comment deleted",
		})
	}
}

// buildCommentTree organizes flat comment list into nested structure
func buildCommentTree(comments []models.Comment) []models.Comment {
	// Create map for quick lookup
	commentMap := make(map[int]*models.Comment)
	for i := range comments {
		commentMap[comments[i].ID] = &comments[i]
		comments[i].Replies = []models.Comment{} // Initialize replies slice
	}

	// Build tree structure
	var roots []models.Comment
	for i := range comments {
		comment := &comments[i]
		if comment.ParentID == nil {
			// Root-level comment
			roots = append(roots, *comment)
		} else {
			// Reply to another comment
			parent, exists := commentMap[*comment.ParentID]
			if exists {
				parent.Replies = append(parent.Replies, *comment)
			}
		}
	}

	return roots
}

// renderCommentsHTML renders comments as HTML for HTMX
func renderCommentsHTML(w http.ResponseWriter, comments []models.Comment) {
	if len(comments) == 0 {
		fmt.Fprintf(w, `<p class="no-comments">No comments yet. Be the first to comment!</p>`)
		return
	}

	for _, comment := range comments {
		renderComment(w, comment, false)
	}
}

// renderComment renders a single comment with its replies
func renderComment(w http.ResponseWriter, comment models.Comment, isReply bool) {
	class := "comment"
	if isReply {
		class += " comment-reply"
	}

	fmt.Fprintf(w, `<div class="%s">`, class)
	fmt.Fprintf(w, `<div class="comment-header">`)
	fmt.Fprintf(w, `<span class="comment-author">%s</span>`, comment.AuthorName)
	fmt.Fprintf(w, `<span class="comment-date">%s</span>`, comment.CreatedAt.Format("January 2, 2006 at 3:04 PM"))
	fmt.Fprintf(w, `</div>`)
	fmt.Fprintf(w, `<div class="comment-content">%s</div>`, comment.Content)

	// Render replies
	for _, reply := range comment.Replies {
		renderComment(w, reply, true)
	}

	fmt.Fprintf(w, `</div>`)
}

