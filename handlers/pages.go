package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func (app *App) HandleHome(w http.ResponseWriter, _ *http.Request) {
	services, _ := app.DB.GetAllServices()
	posts, _ := app.DB.GetAllPosts()

	data := map[string]interface{}{
		"Title":    "Atarnet Homelab - K8s Infrastructure at Home",
		"Services": services[:Min(4, len(services))],
		"Posts":    posts[:Min(3, len(posts))],
	}
	app.Render(w, "home.html", data)
}

func (app *App) HandleServices(w http.ResponseWriter, _ *http.Request) {
	services, _ := app.DB.GetAllServices()

	// Build breadcrumbs
	breadcrumbs := []models.Breadcrumb{
		{Name: "Home", URL: "/"},
		{Name: "Services", URL: ""},
	}

	data := map[string]interface{}{
		"Title":       "Services - Atarnet Homelab",
		"Services":    services,
		"Breadcrumbs": breadcrumbs,
	}
	app.Render(w, "services.html", data)
}

func (app *App) HandleBlog(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	posts, _ := app.DB.GetAllPosts()

	if category != "" {
		filtered := []models.Post{}
		for _, p := range posts {
			if p.Category == category {
				filtered = append(filtered, p)
			}
		}
		posts = filtered
	}

	// Build breadcrumbs
	breadcrumbs := []models.Breadcrumb{
		{Name: "Home", URL: "/"},
		{Name: "Blog", URL: ""},
	}

	data := map[string]interface{}{
		"Title":       "Blog - Atarnet Homelab",
		"Posts":       posts,
		"Category":    category,
		"Breadcrumbs": breadcrumbs,
	}
	app.Render(w, "blog.html", data)
}

func (app *App) HandleBlogPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	post, err := app.DB.GetPostByID(id)
	if err != nil || post == nil {
		app.Handle404(w, r)
		return
	}

	// Increment view count (ignore errors)
	_ = app.DB.IncrementPostViews(id)

	// Build breadcrumbs
	breadcrumbs := []models.Breadcrumb{
		{Name: "Home", URL: "/"},
		{Name: "Blog", URL: "/blog"},
		{Name: post.Title, URL: ""},
	}

	data := map[string]interface{}{
		"Title":       post.Title + " - Atarnet Homelab",
		"Post":        post,
		"Breadcrumbs": breadcrumbs,
	}
	app.Render(w, "post.html", data)
}

func (app *App) HandleAbout(w http.ResponseWriter, _ *http.Request) {
	data := map[string]interface{}{
		"Title": "About - Atarnet Homelab",
	}
	app.Render(w, "about.html", data)
}

func (app *App) HandleAdmin(w http.ResponseWriter, _ *http.Request) {
	posts, _ := app.DB.GetAllPosts()

	data := map[string]interface{}{
		"Title": "Blog Admin - Atarnet Homelab",
		"Posts": posts,
	}
	app.Render(w, "admin.html", data)
}

func (app *App) HandleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if _, err := w.Write([]byte(`{"status":"healthy"}`)); err != nil {
		log.Printf("Error writing health response: %v", err)
	}
}

func (app *App) Handle404(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)

	breadcrumbs := []models.Breadcrumb{
		{Name: "Home", URL: "/"},
		{Name: "Page Not Found", URL: ""},
	}

	data := map[string]interface{}{
		"Title":       "404 - Page Not Found",
		"Breadcrumbs": breadcrumbs,
	}
	app.Render(w, "404.html", data)
}
