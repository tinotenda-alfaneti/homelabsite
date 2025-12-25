package main

import (
	"context"
	"embed"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/tinotenda-alfaneti/homelabsite/config"
	"github.com/tinotenda-alfaneti/homelabsite/db"
	"github.com/tinotenda-alfaneti/homelabsite/handlers"
	"github.com/tinotenda-alfaneti/homelabsite/markdown"
	"github.com/tinotenda-alfaneti/homelabsite/middleware"
	"golang.org/x/time/rate"
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

	// Get database path - smart detection for Kubernetes vs local
	dbPath := config.GetEnv("DB_PATH", "")
	if dbPath == "" {
		// Use /app/data for Kubernetes PVC, local data/ for development
		if _, err := os.Stat("/app/data"); err == nil {
			dbPath = "/app/data/homelab.db"
		} else {
			dbPath = "data/homelab.db"
		}
	}

	log.Printf("Database path: %s", dbPath)

	// Initialize database
	database, err := db.New(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.Close()

	// Check if we need to migrate from YAML
	if shouldMigrate(dbPath) {
		log.Printf("Performing initial migration from YAML to database")
		if err := database.MigrateFromYAML(cfg.Posts, cfg.Services); err != nil {
			log.Fatalf("Failed to migrate from YAML: %v", err)
		}
		// Create migration marker
		markerPath := filepath.Join(filepath.Dir(dbPath), ".migrated")
		os.WriteFile(markerPath, []byte(time.Now().String()), 0644)
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

	// Create rate limiter - 5 requests per second, burst of 10
	rateLimiter := middleware.NewRateLimiter(rate.Limit(5), 10)

	// Create app
	app := &handlers.App{
		Config:     cfg,
		Templates:  templates,
		Auth:       auth,
		ConfigPath: configPath,
		DB:         database,
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
	r.HandleFunc("/search", app.HandleSearchPage).Methods("GET")
	r.HandleFunc("/about", app.HandleAbout).Methods("GET")
	r.HandleFunc("/health", app.HandleHealth).Methods("GET")

	// Auth routes
	r.HandleFunc("/admin/login", app.HandleLoginPage).Methods("GET")
	r.HandleFunc("/admin/login", rateLimiter.RateLimit(app.HandleLogin)).Methods("POST")
	r.HandleFunc("/admin/logout", app.HandleLogout).Methods("GET")
	r.HandleFunc("/admin", auth.RequireAuth(app.HandleAdmin)).Methods("GET")

	// API routes
	r.HandleFunc("/api/services", app.HandleAPIServices).Methods("GET")
	r.HandleFunc("/api/posts", app.HandleAPIPosts).Methods("GET")
	r.HandleFunc("/api/posts/{id}", app.HandleAPIGetPost).Methods("GET")
	r.HandleFunc("/api/posts", auth.RequireAuth(app.HandleAPISavePost)).Methods("POST")
	r.HandleFunc("/api/posts/{id}", auth.RequireAuth(app.HandleAPIDeletePost)).Methods("DELETE")
	r.HandleFunc("/api/search", app.HandleSearch).Methods("GET")
	r.HandleFunc("/api/tags", app.HandleAPITags).Methods("GET")

	// Comment routes
	r.HandleFunc("/api/posts/{id}/comments", handlers.HandleGetComments(database)).Methods("GET")
	r.HandleFunc("/api/posts/{id}/comments", rateLimiter.RateLimit(handlers.HandlePostComment(database))).Methods("POST")
	r.HandleFunc("/api/admin/comments/pending", auth.RequireAuth(handlers.HandleGetPendingComments(database))).Methods("GET")
	r.HandleFunc("/api/admin/comments/{id}/approve", auth.RequireAuth(handlers.HandleApproveComment(database))).Methods("POST")
	r.HandleFunc("/api/admin/comments/{id}", auth.RequireAuth(handlers.HandleDeleteComment(database))).Methods("DELETE")

	// RSS Feed
	r.HandleFunc("/rss", app.HandleRSS).Methods("GET")
	r.HandleFunc("/feed", app.HandleRSS).Methods("GET")

	// Start server with graceful shutdown
	port := config.GetEnv("PORT", "8082")
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Printf("Starting Atarnet Homelab Site on port %s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Println("Shutting down server gracefully...")

	// Create context with timeout for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}

	log.Println("Server stopped")
}

// shouldMigrate checks if we need to migrate from YAML to database
func shouldMigrate(dbPath string) bool {
	markerPath := filepath.Join(filepath.Dir(dbPath), ".migrated")
	_, err := os.Stat(markerPath)
	return os.IsNotExist(err)
}
