package handlers

import (
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/gorilla/mux"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

func (app *App) HandleHome(w http.ResponseWriter, _ *http.Request) {
	projects, _ := app.DB.GetAllProjects()
	posts, _ := app.DB.GetPublishedPosts()

	data := map[string]interface{}{
		"Title":    "Atarnet Homelab - K8s Infrastructure at Home",
		"Projects": projects[:Min(4, len(projects))],
		"Posts":    posts[:Min(3, len(posts))],
	}
	app.Render(w, "home.html", data)
}

func (app *App) HandleProjects(w http.ResponseWriter, r *http.Request) {
	projects, _ := app.DB.GetAllProjects()
	posts, _ := app.DB.GetAllPosts()
	skill := r.URL.Query().Get("skill")

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

	// Build breadcrumbs
	breadcrumbs := []models.Breadcrumb{
		{Name: "Home", URL: "/"},
		{Name: "Projects", URL: "/projects"},
	}
	if skill != "" {
		breadcrumbs = append(breadcrumbs, models.Breadcrumb{Name: skill, URL: ""})
	} else {
		breadcrumbs[len(breadcrumbs)-1].URL = ""
	}

	data := map[string]interface{}{
		"Title":       "Projects - Atarnet Homelab",
		"Projects":    projects,
		"Posts":       posts,
		"Breadcrumbs": breadcrumbs,
		"Skill":       skill,
	}
	app.Render(w, "projects.html", data)
}

func (app *App) HandleBlog(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	posts, _ := app.DB.GetPublishedPosts()

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
	projects, _ := app.DB.GetAllProjects()
	posts, _ := app.DB.GetAllPosts()

	// Aggregate unique skills from projects tech and posts tags
	skillsMap := make(map[string]bool)
	for _, p := range projects {
		techs := strings.Split(p.Tech, " + ")
		for _, tech := range techs {
			tech = strings.TrimSpace(tech)
			if tech != "" {
				skillsMap[tech] = true
			}
		}
	}
	for _, p := range posts {
		for _, tag := range p.Tags {
			tag = strings.TrimSpace(tag)
			if tag != "" {
				skillsMap[tag] = true
			}
		}
	}
	skills := []string{}
	for skill := range skillsMap {
		skills = append(skills, skill)
	}
	sort.Strings(skills)

	data := map[string]interface{}{
		"Title":  "About - Atarnet Homelab",
		"Skills": skills,
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
