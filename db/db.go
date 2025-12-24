package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("creating db directory: %w", err)
	}

	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Set connection pool settings
	conn.SetMaxOpenConns(1) // SQLite works best with single connection
	conn.SetMaxIdleConns(1)
	conn.SetConnMaxLifetime(0)

	// Test connection
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	db := &DB{conn: conn}

	// Initialize schema
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("initializing schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates the database schema if it doesn't exist
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS posts (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		date DATETIME NOT NULL,
		category TEXT NOT NULL,
		summary TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_posts_date ON posts(date DESC);
	CREATE INDEX IF NOT EXISTS idx_posts_category ON posts(category);

	CREATE TABLE IF NOT EXISTS services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		description TEXT NOT NULL,
		url TEXT NOT NULL,
		tech TEXT NOT NULL,
		status TEXT NOT NULL,
		icon TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_services_status ON services(status);
	`

	_, err := db.conn.Exec(schema)
	return err
}

// GetAllPosts retrieves all posts from the database
func (db *DB) GetAllPosts() ([]models.Post, error) {
	query := `SELECT id, title, date, category, summary, content, tags FROM posts ORDER BY date DESC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var tags string
		if err := rows.Scan(&p.ID, &p.Title, &p.Date, &p.Category, &p.Summary, &p.Content, &tags); err != nil {
			return nil, err
		}
		// Parse tags from comma-separated string
		if tags != "" {
			p.Tags = parseTagsFromString(tags)
		}
		posts = append(posts, p)
	}

	return posts, rows.Err()
}

// GetPostByID retrieves a single post by ID
func (db *DB) GetPostByID(id string) (*models.Post, error) {
	query := `SELECT id, title, date, category, summary, content, tags FROM posts WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var p models.Post
	var tags string
	if err := row.Scan(&p.ID, &p.Title, &p.Date, &p.Category, &p.Summary, &p.Content, &tags); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	if tags != "" {
		p.Tags = parseTagsFromString(tags)
	}

	return &p, nil
}

// SavePost creates or updates a post
func (db *DB) SavePost(post *models.Post) error {
	tags := joinTags(post.Tags)

	query := `
	INSERT INTO posts (id, title, date, category, summary, content, tags, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(id) DO UPDATE SET
		title = excluded.title,
		date = excluded.date,
		category = excluded.category,
		summary = excluded.summary,
		content = excluded.content,
		tags = excluded.tags,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.conn.Exec(query, post.ID, post.Title, post.Date, post.Category, post.Summary, post.Content, tags)
	return err
}

// DeletePost deletes a post by ID
func (db *DB) DeletePost(id string) error {
	query := `DELETE FROM posts WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// GetAllServices retrieves all services from the database
func (db *DB) GetAllServices() ([]models.Service, error) {
	query := `SELECT name, description, url, tech, status, icon FROM services ORDER BY name`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []models.Service
	for rows.Next() {
		var s models.Service
		if err := rows.Scan(&s.Name, &s.Description, &s.URL, &s.Tech, &s.Status, &s.Icon); err != nil {
			return nil, err
		}
		services = append(services, s)
	}

	return services, rows.Err()
}

// SaveService creates or updates a service
func (db *DB) SaveService(service *models.Service) error {
	// Check if service exists by name
	var exists bool
	err := db.conn.QueryRow(`SELECT EXISTS(SELECT 1 FROM services WHERE name = ?)`, service.Name).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		query := `
		UPDATE services 
		SET description = ?, url = ?, tech = ?, status = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
		WHERE name = ?
		`
		_, err = db.conn.Exec(query, service.Description, service.URL, service.Tech, service.Status, service.Icon, service.Name)
	} else {
		query := `
		INSERT INTO services (name, description, url, tech, status, icon)
		VALUES (?, ?, ?, ?, ?, ?)
		`
		_, err = db.conn.Exec(query, service.Name, service.Description, service.URL, service.Tech, service.Status, service.Icon)
	}

	return err
}

// MigrateFromYAML imports data from YAML files into the database
func (db *DB) MigrateFromYAML(posts []models.Post, services []models.Service) error {
	log.Printf("Migrating %d posts and %d services from YAML to database", len(posts), len(services))

	// Start transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Migrate posts
	for _, post := range posts {
		if post.Date.IsZero() {
			post.Date = time.Now()
		}
		if err := db.SavePost(&post); err != nil {
			return fmt.Errorf("migrating post %s: %w", post.ID, err)
		}
	}

	// Migrate services
	for _, service := range services {
		if err := db.SaveService(&service); err != nil {
			return fmt.Errorf("migrating service %s: %w", service.Name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	log.Printf("Migration completed successfully")
	return nil
}

// Helper functions
func parseTagsFromString(tags string) []string {
	if tags == "" {
		return []string{}
	}
	// Simple comma-separated parsing
	result := []string{}
	current := ""
	for _, c := range tags {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func joinTags(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	result := ""
	for i, tag := range tags {
		if i > 0 {
			result += ","
		}
		result += tag
	}
	return result
}
