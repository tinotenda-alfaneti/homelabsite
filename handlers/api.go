package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

const htmxRequestHeader = "true"

func (app *App) HandleAPIServices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")

	// Get services from database
	services, err := app.DB.GetAllServices()
	if err != nil {
		log.Printf("Error getting services from database: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if status != "" {
		filtered := []models.Service{}
		for _, s := range services {
			if s.Status == status {
				filtered = append(filtered, s)
			}
		}
		services = filtered
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == htmxRequestHeader {
		// Return HTML fragment
		data := map[string]interface{}{
			"Services": services,
		}
		w.Header().Set("Content-Type", "text/html")
		if err := app.Templates.ExecuteTemplate(w, "services-grid", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return JSON for non-HTMX requests
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(services); err != nil {
		log.Printf("Error encoding services to JSON: %v", err)
	}
}

func (app *App) HandleAPIPosts(w http.ResponseWriter, _ *http.Request) {
	posts, err := app.DB.GetAllPosts()
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
	var post models.Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
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
