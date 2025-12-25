package handlers

import (
	"encoding/json"
	"log"
	"net/http"
)

func (app *App) HandleSearch(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")

	var posts interface{}
	var err error

	if tag != "" {
		// Search by tag
		posts, err = app.DB.SearchPostsByTag(tag)
	} else if query != "" {
		// Full-text search
		posts, err = app.DB.SearchPosts(query)
	} else {
		// No search query, return empty
		posts, err = app.DB.GetAllPosts()
	}

	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Check if HTMX request
	if r.Header.Get("HX-Request") == "true" {
		data := map[string]interface{}{
			"Posts": posts,
			"Query": query,
			"Tag":   tag,
		}
		w.Header().Set("Content-Type", "text/html")
		if err := app.Templates.ExecuteTemplate(w, "blog-posts", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"posts": posts,
		"query": query,
		"tag":   tag,
	})
}

func (app *App) HandleSearchPage(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	tag := r.URL.Query().Get("tag")

	var posts interface{}
	var err error
	var searchType string

	if tag != "" {
		posts, err = app.DB.SearchPostsByTag(tag)
		searchType = "tag"
	} else if query != "" {
		posts, err = app.DB.SearchPosts(query)
		searchType = "query"
	} else {
		posts, err = app.DB.GetAllPosts()
		searchType = "all"
	}

	if err != nil {
		log.Printf("Search error: %v", err)
		http.Error(w, "Search failed", http.StatusInternalServerError)
		return
	}

	// Get all tags for the sidebar/filter
	allTags, _ := app.DB.GetAllTags()

	data := map[string]interface{}{
		"Title":      "Search - Atarnet Homelab",
		"Posts":      posts,
		"Query":      query,
		"Tag":        tag,
		"SearchType": searchType,
		"AllTags":    allTags,
	}

	app.Render(w, "search.html", data)
}

func (app *App) HandleAPITags(w http.ResponseWriter, r *http.Request) {
	tags, err := app.DB.GetAllTags()
	if err != nil {
		log.Printf("Error getting tags: %v", err)
		http.Error(w, "Failed to get tags", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tags": tags,
	})
}
