package main

import (
	"crypto/rand"
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

//go:embed web/templates/* web/static/*
var embedFS embed.FS

type Config struct {
	Services []Service `yaml:"services"`
	Posts    []Post    `yaml:"posts"`
}

type Service struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description" json:"description"`
	URL         string `yaml:"url" json:"url"`
	Tech        string `yaml:"tech" json:"tech"`
	Status      string `yaml:"status" json:"status"`
	Icon        string `yaml:"icon" json:"icon"`
}

type Post struct {
	ID       string    `yaml:"id" json:"id"`
	Title    string    `yaml:"title" json:"title"`
	Date     time.Time `yaml:"date" json:"date"`
	Category string    `yaml:"category" json:"category"`
	Summary  string    `yaml:"summary" json:"summary"`
	Content  string    `yaml:"content" json:"content"`
	Tags     []string  `yaml:"tags" json:"tags"`
}

type App struct {
	config    Config
	templates *template.Template
	sessions  map[string]time.Time
	sessionMu sync.RWMutex
	adminUser string
	adminPass string
}

func main() {
	// Load .env file if it exists (ignore error if not found - we'll use env vars or defaults)
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables or defaults")
	}

	app := &App{
		sessions:  make(map[string]time.Time),
		adminUser: getEnv("ADMIN_USER", "admin"),
		adminPass: getEnv("ADMIN_PASS", "changeme"),
	}

	log.Printf("Admin credentials - User: %s (set via ADMIN_USER and ADMIN_PASS in .env or env vars)", app.adminUser)

	// Load configuration
	if err := app.loadConfig(); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Parse templates
	if err := app.loadTemplates(); err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Start session cleanup
	go app.cleanupSessions()

	// Setup router
	r := mux.NewRouter()

	// Static files - serve from web/static with /static prefix stripped
	staticFS, err := fs.Sub(embedFS, "web/static")
	if err != nil {
		log.Fatalf("Failed to create static filesystem: %v", err)
	}
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Routes
	r.HandleFunc("/", app.handleHome).Methods("GET")
	r.HandleFunc("/services", app.handleServices).Methods("GET")
	r.HandleFunc("/blog", app.handleBlog).Methods("GET")
	r.HandleFunc("/blog/{id}", app.handleBlogPost).Methods("GET")
	r.HandleFunc("/about", app.handleAbout).Methods("GET")
	r.HandleFunc("/admin/login", app.handleLoginPage).Methods("GET")
	r.HandleFunc("/admin/login", app.handleLogin).Methods("POST")
	r.HandleFunc("/admin/logout", app.handleLogout).Methods("GET")
	r.HandleFunc("/admin", app.requireAuth(app.handleAdmin)).Methods("GET")
	r.HandleFunc("/health", app.handleHealth).Methods("GET")

	// API endpoints - protected
	r.HandleFunc("/api/services", app.handleAPIServices).Methods("GET")
	r.HandleFunc("/api/posts", app.handleAPIPosts).Methods("GET")
	r.HandleFunc("/api/posts/{id}", app.handleAPIGetPost).Methods("GET")
	r.HandleFunc("/api/posts", app.requireAuth(app.handleAPISavePost)).Methods("POST")
	r.HandleFunc("/api/posts/{id}", app.requireAuth(app.handleAPIDeletePost)).Methods("DELETE")

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Starting Atarnet Homelab Site on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func (app *App) loadConfig() error {
	data, err := os.ReadFile("config/config.yaml")
	if err != nil {
		return fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, &app.config); err != nil {
		return fmt.Errorf("parsing config: %w", err)
	}

	// Sort posts by date descending
	sort.Slice(app.config.Posts, func(i, j int) bool {
		return app.config.Posts[i].Date.After(app.config.Posts[j].Date)
	})

	return nil
}

func (app *App) loadTemplates() error {
	var err error
	funcMap := template.FuncMap{
		"markdown": func(content string) template.HTML {
			return template.HTML(renderMarkdown(content))
		},
	}
	app.templates, err = template.New("").Funcs(funcMap).ParseFS(embedFS, "web/templates/*.html")
	return err
}

func renderMarkdown(content string) string {
	// Simple markdown rendering - headers, bold, lists, code blocks
	lines := strings.Split(content, "\n")
	var html strings.Builder
	inCodeBlock := false
	inList := false

	for i, line := range lines {
		// Code blocks
		if strings.HasPrefix(line, "```") {
			if inCodeBlock {
				html.WriteString("</code></pre>")
				inCodeBlock = false
			} else {
				html.WriteString("<pre><code>")
				inCodeBlock = true
			}
			continue
		}

		if inCodeBlock {
			html.WriteString(template.HTMLEscapeString(line) + "\n")
			continue
		}

		// Headers
		if strings.HasPrefix(line, "## ") {
			if inList {
				html.WriteString("</ul>")
				inList = false
			}
			html.WriteString("<h2>" + template.HTMLEscapeString(strings.TrimPrefix(line, "## ")) + "</h2>")
			continue
		}
		if strings.HasPrefix(line, "### ") {
			if inList {
				html.WriteString("</ul>")
				inList = false
			}
			html.WriteString("<h3>" + template.HTMLEscapeString(strings.TrimPrefix(line, "### ")) + "</h3>")
			continue
		}

		// Lists
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") {
			if !inList {
				html.WriteString("<ul>")
				inList = true
			}
			text := strings.TrimPrefix(strings.TrimPrefix(line, "- "), "* ")
			text = processBold(text)
			html.WriteString("<li>" + text + "</li>")
			continue
		}

		// Numbered lists
		if len(line) > 2 && line[0] >= '0' && line[0] <= '9' && line[1] == '.' {
			if !inList {
				html.WriteString("<ol>")
				inList = true
			}
			parts := strings.SplitN(line, ". ", 2)
			if len(parts) == 2 {
				text := processBold(parts[1])
				html.WriteString("<li>" + text + "</li>")
			}
			continue
		}

		// End list if we hit a non-list line
		if inList && line != "" && !strings.HasPrefix(line, "- ") && !strings.HasPrefix(line, "* ") {
			html.WriteString("</ul>")
			inList = false
		}

		// Empty lines
		if strings.TrimSpace(line) == "" {
			if i > 0 && i < len(lines)-1 {
				html.WriteString("<br>")
			}
			continue
		}

		// Regular paragraphs
		if !inList {
			text := processBold(line)
			html.WriteString("<p>" + text + "</p>")
		}
	}

	if inList {
		html.WriteString("</ul>")
	}
	if inCodeBlock {
		html.WriteString("</code></pre>")
	}

	return html.String()
}

func processBold(text string) string {
	// First escape the text
	escaped := template.HTMLEscapeString(text)
	// Then handle **bold** text
	for strings.Contains(escaped, "**") {
		first := strings.Index(escaped, "**")
		if first == -1 {
			break
		}
		second := strings.Index(escaped[first+2:], "**")
		if second == -1 {
			break
		}
		second += first + 2
		escaped = escaped[:first] + "<strong>" + escaped[first+2:second] + "</strong>" + escaped[second+2:]
	}
	return escaped
}

func (app *App) handleHome(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Atarnet Homelab - Production Infrastructure at Home",
		"Services": app.config.Services[:min(4, len(app.config.Services))],
		"Posts":    app.config.Posts[:min(3, len(app.config.Posts))],
	}
	app.render(w, "home.html", data)
}

func (app *App) handleServices(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title":    "Services - Atarnet Homelab",
		"Services": app.config.Services,
	}
	app.render(w, "services.html", data)
}

func (app *App) handleBlog(w http.ResponseWriter, r *http.Request) {
	category := r.URL.Query().Get("category")
	posts := app.config.Posts

	if category != "" {
		filtered := []Post{}
		for _, p := range app.config.Posts {
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
	app.render(w, "blog.html", data)
}

func (app *App) handleBlogPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var post *Post
	for i := range app.config.Posts {
		if app.config.Posts[i].ID == id {
			post = &app.config.Posts[i]
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
	app.render(w, "post.html", data)
}

func (app *App) handleAbout(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "About - Atarnet Homelab",
	}
	app.render(w, "about.html", data)
}

func (app *App) handleAdmin(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Blog Admin - Atarnet Homelab",
		"Posts": app.config.Posts,
	}
	app.render(w, "admin.html", data)
}

func (app *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
}

func (app *App) handleAPIServices(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	services := app.config.Services

	if status != "" {
		filtered := []Service{}
		for _, s := range app.config.Services {
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
		if err := app.templates.ExecuteTemplate(w, "services-grid", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Return JSON for non-HTMX requests
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(services)
}

func (app *App) handleAPIPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(app.config.Posts)
}

func (app *App) handleAPIGetPost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	for _, post := range app.config.Posts {
		if post.ID == id {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(post)
			return
		}
	}

	http.NotFound(w, r)
}

func (app *App) handleAPISavePost(w http.ResponseWriter, r *http.Request) {
	var post Post
	if err := json.NewDecoder(r.Body).Decode(&post); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Find and update existing post or add new one
	found := false
	for i := range app.config.Posts {
		if app.config.Posts[i].ID == post.ID {
			app.config.Posts[i] = post
			found = true
			break
		}
	}

	if !found {
		app.config.Posts = append(app.config.Posts, post)
	}

	// Sort posts by date descending
	sort.Slice(app.config.Posts, func(i, j int) bool {
		return app.config.Posts[i].Date.After(app.config.Posts[j].Date)
	})

	// Save to config file
	if err := app.saveConfig(); err != nil {
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

func (app *App) handleAPIDeletePost(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Find and remove post
	for i, post := range app.config.Posts {
		if post.ID == id {
			app.config.Posts = append(app.config.Posts[:i], app.config.Posts[i+1:]...)

			// Save to config file
			if err := app.saveConfig(); err != nil {
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

func (app *App) saveConfig() error {
	data, err := yaml.Marshal(&app.config)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile("config/config.yaml", data, 0644)
}

func (app *App) render(w http.ResponseWriter, tmpl string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// First load the specific page template to define the content block
	if err := app.templates.ExecuteTemplate(w, tmpl, data); err != nil {
		log.Printf("Error rendering template %s: %v", tmpl, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// Authentication middleware
func (app *App) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		app.sessionMu.RLock()
		expiry, exists := app.sessions[cookie.Value]
		app.sessionMu.RUnlock()

		if !exists || time.Now().After(expiry) {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}

		// Extend session
		app.sessionMu.Lock()
		app.sessions[cookie.Value] = time.Now().Add(24 * time.Hour)
		app.sessionMu.Unlock()

		next(w, r)
	}
}

func (app *App) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Admin Login - Atarnet Homelab",
		"Error": nil,
	}
	app.render(w, "login.html", data)
}

func (app *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if username != app.adminUser || password != app.adminPass {
		data := map[string]interface{}{
			"Title": "Admin Login - Atarnet Homelab",
			"Error": "Invalid username or password",
		}
		app.render(w, "login.html", data)
		return
	}

	// Create session
	sessionToken := generateSessionToken()
	app.sessionMu.Lock()
	app.sessions[sessionToken] = time.Now().Add(24 * time.Hour)
	app.sessionMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (app *App) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		app.sessionMu.Lock()
		delete(app.sessions, cookie.Value)
		app.sessionMu.Unlock()
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/", http.StatusFound)
}

func (app *App) cleanupSessions() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		app.sessionMu.Lock()
		now := time.Now()
		for token, expiry := range app.sessions {
			if now.After(expiry) {
				delete(app.sessions, token)
			}
		}
		app.sessionMu.Unlock()
	}
}

func generateSessionToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
