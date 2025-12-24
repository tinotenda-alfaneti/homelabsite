package handlers

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func (app *App) HandleHome(w http.ResponseWriter, r *http.Request) {
	services, _ := app.DB.GetAllServices()
	posts, _ := app.DB.GetAllPosts()
	
	data := map[string]interface{}{
		"Title":    "Atarnet Homelab - K8s Infrastructure at Home",
		"Services": services[:Min(4, len(services))],
		"Posts":    posts[:Min(3, len(posts))],
	}
	app.Render(w, "home.html", data)
}

func (app *App) HandleServices(w http.ResponseWriter, r *http.Request) {
	services, _ := app.DB.GetAllServices()
	
	data := map[string]interface{}{
		"Title":    "Services - Atarnet Homelab",
		"Services": services,
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

	post, err := app.DB.GetPostByID(id)
	if err != nil || post == nil {
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
	posts, _ := app.DB.GetAllPosts()
	
	data := map[string]interface{}{
		"Title": "Blog Admin - Atarnet Homelab",
		"Posts": posts,
	}
	app.Render(w, "admin.html", data)
}

func (app *App) HandleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"healthy"}`))
}
