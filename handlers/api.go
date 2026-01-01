package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

const htmxRequestHeader = "true"

func (app *App) HandleAPIProjects(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	skill := r.URL.Query().Get("skill")

	// Get projects from database
	projects, err := app.DB.GetAllProjects()
	if err != nil {
		log.Printf("Error getting projects from database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Get posts from database
	posts, err := app.DB.GetAllPosts()
	if err != nil {
		log.Printf("Error getting posts from database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if status != "" {
		filtered := []models.Project{}
		for _, p := range projects {
			if p.Status == status {
				filtered = append(filtered, p)
			}
		}
		projects = filtered
	}

	if skill != "" {
		filteredProjects := []models.Project{}
		for _, p := range projects {
			techs := strings.Split(p.Tech, " + ")
			for _, tech := range techs {
				tech = strings.TrimSpace(tech)
				if tech == skill {
					filteredProjects = append(filteredProjects, p)
					break
				}
			}
		}
		projects = filteredProjects

		filteredPosts := []models.Post{}
		for _, p := range posts {
			for _, tag := range p.Tags {
				tag = strings.TrimSpace(tag)
				if tag == skill {
					filteredPosts = append(filteredPosts, p)
					break
				}
			}
		}
		posts = filteredPosts
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == htmxRequestHeader {
		// Return HTML fragment
		data := map[string]interface{}{
			"Projects": projects,
			"Posts":    posts,
		}
		w.Header().Set("Content-Type", "text/html")
		if err := app.Templates.ExecuteTemplate(w, "projects-grid", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return JSON for non-HTMX requests
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(projects); err != nil {
		log.Printf("Error encoding projects to JSON: %v", err)
	}
}

func (app *App) HandleAPIPosts(w http.ResponseWriter, _ *http.Request) {
	posts, err := app.DB.GetPublishedPosts()
	if err != nil {
		log.Printf("Error getting posts from database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		log.Printf("Error encoding posts to JSON: %v", err)
	}
}

func (app *App) HandleAPIGetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	post, err := app.DB.GetPostByID(id)
	if err != nil {
		log.Printf("Error getting post from database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if post == nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(post); err != nil {
		log.Printf("Error encoding post to JSON: %v", err)
	}
}

func (app *App) HandleAPISavePost(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form
	err := r.ParseMultipartForm(32 << 20) // 32MB max
	if err != nil {
		http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
		return
	}

	var post models.Post
	post.ID = r.FormValue("id")
	if post.ID == "" {
		post.ID = "post-" + fmt.Sprintf("%d", time.Now().Unix())
	}
	post.Title = r.FormValue("title")
	post.Category = r.FormValue("category")
	post.Summary = r.FormValue("summary")
	post.Content = r.FormValue("content")

	// Parse tags
	tagsStr := r.FormValue("tags")
	if tagsStr != "" {
		if err := json.Unmarshal([]byte(tagsStr), &post.Tags); err != nil {
			log.Printf("Error parsing tags: %v", err)
			post.Tags = []string{}
		}
	}

	// Parse date
	dateStr := r.FormValue("date")
	if dateStr != "" {
		if parsed, err := time.Parse(time.RFC3339, dateStr); err == nil {
			post.Date = parsed
		} else {
			post.Date = time.Now()
		}
	} else {
		post.Date = time.Now()
	}

	// Parse status
	post.Status = r.FormValue("status")
	if post.Status == "" {
		post.Status = "published"
	}

	// Handle file upload
	file, header, err := r.FormFile("image")
	if err == nil && header != nil {
		defer file.Close()

		// Validate file type
		contentType := header.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") && !strings.HasPrefix(contentType, "video/") && !strings.HasPrefix(contentType, "audio/") {
			http.Error(w, "Invalid file type. Only images, videos, and audio files are allowed.", http.StatusBadRequest)
			return
		}

		// Validate file size (10MB max)
		if header.Size > 10<<20 {
			http.Error(w, "File too large. Maximum size is 10MB.", http.StatusBadRequest)
			return
		}

		// Generate unique filename
		ext := filepath.Ext(header.Filename)
		filename := fmt.Sprintf("%s_%d%s", post.ID, time.Now().Unix(), ext)
		filePath := filepath.Join("web/static/uploads", filename)

		// Create uploads directory if it doesn't exist
		if err := os.MkdirAll("web/static/uploads", 0755); err != nil {
			log.Printf("Error creating uploads directory: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Save file
		dst, err := os.Create(filePath)
		if err != nil {
			log.Printf("Error creating file: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		if _, err := io.Copy(dst, file); err != nil {
			log.Printf("Error saving file: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		// Optionally, add image reference to content if it's an image
		if strings.HasPrefix(contentType, "image/") {
			imageMarkdown := fmt.Sprintf("\n\n![%s](/uploads/%s)\n\n", header.Filename, filename)
			post.Content += imageMarkdown
		}
	} else if err != http.ErrMissingFile {
		log.Printf("Error reading file: %v", err)
	}

	// Save to database
	if err := app.DB.SavePost(&post); err != nil {
		log.Printf("Error saving post: %v", err)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"post":    post,
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (app *App) HandleAPIDeletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Delete from database
	if err := app.DB.DeletePost(id); err != nil {
		log.Printf("Error deleting post: %v", err)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}); err != nil {
			log.Printf("Error encoding error response: %v", err)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
	}); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

func (app *App) HandleAPIPopularPosts(w http.ResponseWriter, _ *http.Request) {
	limit := 5 // Default to 5 popular posts

	posts, err := app.DB.GetPopularPosts(limit)
	if err != nil {
		log.Printf("Error getting popular posts: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(posts); err != nil {
		log.Printf("Error encoding popular posts to JSON: %v", err)
	}
}
