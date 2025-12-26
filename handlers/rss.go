package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gorilla/feeds"
)

func (app *App) HandleRSS(w http.ResponseWriter, r *http.Request) {
	// Get posts from database
	posts, err := app.DB.GetAllPosts()
	if err != nil {
		log.Printf("Error getting posts for RSS: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Create feed
	now := time.Now()
	feed := &feeds.Feed{
		Title:       "Atarnet Homelab Blog",
		Link:        &feeds.Link{Href: "https://atarnet.org/blog"},
		Description: "Deep-dives into Kubernetes, Go, DevOps, infrastructure, and system architecture",
		Author:      &feeds.Author{Name: "Tinotenda Alfaneti", Email: "tinotenda@atarnet.org"},
		Created:     now,
	}

	// Add posts as feed items
	feed.Items = make([]*feeds.Item, 0, len(posts))
	for _, post := range posts {
		item := &feeds.Item{
			Title:       post.Title,
			Link:        &feeds.Link{Href: "https://atarnet.org/blog/" + post.ID},
			Description: post.Summary,
			Author:      &feeds.Author{Name: "Tinotenda Alfaneti", Email: "tinotenda@atarnet.org"},
			Created:     post.Date,
			Content:     post.Content,
		}

		feed.Items = append(feed.Items, item)
	}

	// Determine format from URL path or Accept header
	format := "rss"
	if r.URL.Path == "/atom" || r.Header.Get("Accept") == "application/atom+xml" {
		format = "atom"
	}

	// Generate feed
	var feedContent string
	var contentType string

	switch format {
	case "atom":
		atom, err := feed.ToAtom()
		if err != nil {
			log.Printf("Error generating Atom feed: %v", err)
			http.Error(w, "Error generating feed", http.StatusInternalServerError)
			return
		}
		feedContent = atom
		contentType = "application/atom+xml"
	default:
		rss, err := feed.ToRss()
		if err != nil {
			log.Printf("Error generating RSS feed: %v", err)
			http.Error(w, "Error generating feed", http.StatusInternalServerError)
			return
		}
		feedContent = rss
		contentType = "application/rss+xml"
	}

	w.Header().Set("Content-Type", contentType+"; charset=utf-8")
	if _, err := w.Write([]byte(feedContent)); err != nil {
		log.Printf("Error writing RSS feed: %v", err)
	}
}
