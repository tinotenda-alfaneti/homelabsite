package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func (app *App) HandleAPIServices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	services := app.Config.Services

	if status != "" {
		filtered := []models.Service{}
		for _, s := range app.Config.Services {
			if s.Status == status {
				filtered = append(filtered, s)
			}
		}
		services = filtered
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
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
	json.NewEncoder(w).Encode(services)
}

func (app *App) HandleAPIPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app.Config.Posts)
}

func (app *App) HandleAPIGetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, post := range app.Config.Posts {
		if post.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(post)
			return
		}
	}

	http.NotFound(w, r)
}

func (app *App) HandleAPISavePost(w http.ResponseWriter, r *http.Request) {
	var post models.Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find and update existing post or add new one
	found := false
	for i := range app.Config.Posts {
		if app.Config.Posts[i].ID == post.ID {
			app.Config.Posts[i] = post
			found = true
			break
		}
	}

	if !found {
		app.Config.Posts = append(app.Config.Posts, post)
	}

	// Sort posts by date descending
	sort.Slice(app.Config.Posts, func(i, j int) bool {
		return app.Config.Posts[i].Date.After(app.Config.Posts[j].Date)
	})

	// Save to config file
	if err := app.SaveConfig(); err != nil {
		log.Printf("Error saving config: %v", err)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"post":    post,
	})
}

func (app *App) HandleAPIDeletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Find and remove post
	for i, post := range app.Config.Posts {
		if post.ID == id {
			app.Config.Posts = append(app.Config.Posts[:i], app.Config.Posts[i+1:]...)

			// Save to config file
			if err := app.SaveConfig(); err != nil {
				log.Printf("Error saving config: %v", err)
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"error":   err.Error(),
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
			})
			return
		}
	}

	http.NotFound(w, r)
}
