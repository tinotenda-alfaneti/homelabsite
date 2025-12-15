package main

import (
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/tinotenda-alfaneti/homelabsite/config"
	"github.com/tinotenda-alfaneti/homelabsite/handlers"
	"github.com/tinotenda-alfaneti/homelabsite/markdown"
	"github.com/tinotenda-alfaneti/homelabsite/middleware"
)

//go:embed web/templates/* web/static/*
var embedFS embed.FS

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Printf("No .env file found, using environment variables or defaults")
	}

	// Get admin credentials from environment (Kubernetes Secret or local .env)
	adminUser := config.GetEnv("ADMIN_USER", "admin")
	adminPass := config.GetEnv("ADMIN_PASS", "changeme")

	// Get config path - smart detection for Kubernetes vs local
	configPath := config.GetConfigPath()

	log.Printf("Admin credentials - User: %s (set via ADMIN_USER and ADMIN_PASS)", adminUser)
	log.Printf("Config path: %s", configPath)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Parse templates
	funcMap := template.FuncMap{
		"markdown": markdown.Render,
	}
	templates, err := template.New("").Funcs(funcMap).ParseFS(embedFS, "web/templates/*.html")
	if err != nil {
		log.Fatalf("Failed to load templates: %v", err)
	}

	// Create auth middleware
	auth := middleware.NewAuthMiddleware(adminUser, adminPass)

	// Create app
	app := &handlers.App{
		Config:     cfg,
		Templates:  templates,
		Auth:       auth,
		ConfigPath: configPath,
	}

	// Setup router
	r := mux.NewRouter()

	// Static files
	staticFS, err := fs.Sub(embedFS, "web/static")
	if err != nil {
		log.Fatalf("Failed to create static filesystem: %v", err)
	}
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Page routes
	r.HandleFunc("/", app.HandleHome).Methods("GET")
	r.HandleFunc("/services", app.HandleServices).Methods("GET")
	r.HandleFunc("/blog", app.HandleBlog).Methods("GET")
	r.HandleFunc("/blog/{id}", app.HandleBlogPost).Methods("GET")
	r.HandleFunc("/about", app.HandleAbout).Methods("GET")
	r.HandleFunc("/health", app.HandleHealth).Methods("GET")

	// Auth routes
	r.HandleFunc("/admin/login", app.HandleLoginPage).Methods("GET")
	r.HandleFunc("/admin/login", app.HandleLogin).Methods("POST")
	r.HandleFunc("/admin/logout", app.HandleLogout).Methods("GET")
	r.HandleFunc("/admin", auth.RequireAuth(app.HandleAdmin)).Methods("GET")

	// API routes
	r.HandleFunc("/api/services", app.HandleAPIServices).Methods("GET")
	r.HandleFunc("/api/posts", app.HandleAPIPosts).Methods("GET")
	r.HandleFunc("/api/posts/{id}", app.HandleAPIGetPost).Methods("GET")
	r.HandleFunc("/api/posts", auth.RequireAuth(app.HandleAPISavePost)).Methods("POST")
	r.HandleFunc("/api/posts/{id}", auth.RequireAuth(app.HandleAPIDeletePost)).Methods("DELETE")

	// Start server
	port := config.GetEnv("PORT", "8082")
	log.Printf("Starting Atarnet Homelab Site on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
