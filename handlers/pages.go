package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func (app *App) HandleHome(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Atarnet Homelab - K8s Infrastructure at Home",
		"Services": app.Config.Services[:Min(4, len(app.Config.Services))],
		"Posts":    app.Config.Posts[:Min(3, len(app.Config.Posts))],
	}
	app.Render(w, "home.html", data)
}

func (app *App) HandleServices(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Services - Atarnet Homelab",
		"Services": app.Config.Services,
	}
	app.Render(w, "services.html", data)
}

func (app *App) HandleBlog(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	posts := app.Config.Posts

	if category != "" {
		filtered := []models.Post{}
		for _, p := range app.Config.Posts {
			if p.Category == category {
				filtered = append(filtered, p)
			}
		}
		posts = filtered
	}

	data := map[string]interface{}{
		"Title":    "Blog - Atarnet Homelab",
		"Posts":    posts,
		"Category": category,
	}
	app.Render(w, "blog.html", data)
}

func (app *App) HandleBlogPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var post *models.Post
	for i := range app.Config.Posts {
		if app.Config.Posts[i].ID == id {
			post = &app.Config.Posts[i]
			break
		}
	}

	if post == nil {
		http.NotFound(w, r)
		return
	}

	data := map[string]interface{}{
		"Title": post.Title + " - Atarnet Homelab",
		"Post":  post,
	}
	app.Render(w, "post.html", data)
}

func (app *App) HandleAbout(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "About - Atarnet Homelab",
	}
	app.Render(w, "about.html", data)
}

func (app *App) HandleAdmin(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Blog Admin - Atarnet Homelab",
		"Posts": app.Config.Posts,
	}
	app.Render(w, "admin.html", data)
}

func (app *App) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"healthy"}`))
}
