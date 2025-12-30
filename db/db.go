package db

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	// Import SQLite driver for database/sql registration
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

// GetConn returns the underlying database connection
func (db *DB) GetConn() *sql.DB {
	return db.conn
}

// initSchema creates the database schema if it doesn't exist
func (db *DB) initSchema() error {
	// Enable foreign key constraints
	if _, err := db.conn.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return fmt.Errorf("enabling foreign keys: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS posts (
		id TEXT PRIMARY KEY,
		title TEXT NOT NULL,
		date DATETIME NOT NULL,
		category TEXT NOT NULL,
		summary TEXT NOT NULL,
		content TEXT NOT NULL,
		tags TEXT NOT NULL,
		views INTEGER DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_posts_date ON posts(date DESC);
	CREATE INDEX IF NOT EXISTS idx_posts_category ON posts(category);
	CREATE INDEX IF NOT EXISTS idx_posts_views ON posts(views DESC);

	CREATE TABLE IF NOT EXISTS projects (
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

	CREATE INDEX IF NOT EXISTS idx_projects_status ON projects(status);
	`

	if _, err := db.conn.Exec(schema); err != nil {
		return err
	}

	// Create comments table
	if err := CreateCommentsTable(db.conn); err != nil {
		return fmt.Errorf("creating comments table: %w", err)
	}

	return nil
}

// GetAllPosts retrieves all posts from the database
func (db *DB) GetAllPosts() ([]models.Post, error) {
	query := `SELECT id, title, date, category, summary, content, tags, COALESCE(views, 0) FROM posts ORDER BY date DESC`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var tags string
		if err := rows.Scan(&p.ID, &p.Title, &p.Date, &p.Category, &p.Summary, &p.Content, &tags, &p.Views); err != nil {
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
	query := `SELECT id, title, date, category, summary, content, tags, COALESCE(views, 0) FROM posts WHERE id = ?`
	row := db.conn.QueryRow(query, id)

	var p models.Post
	var tags string
	if err := row.Scan(&p.ID, &p.Title, &p.Date, &p.Category, &p.Summary, &p.Content, &tags, &p.Views); err != nil {
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
	INSERT INTO posts (id, title, date, category, summary, content, tags, views, updated_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(id) DO UPDATE SET
		title = excluded.title,
		date = excluded.date,
		category = excluded.category,
		summary = excluded.summary,
		content = excluded.content,
		tags = excluded.tags,
		updated_at = CURRENT_TIMESTAMP
	`

	_, err := db.conn.Exec(query, post.ID, post.Title, post.Date, post.Category, post.Summary, post.Content, tags, post.Views)
	return err
}

// DeletePost deletes a post by ID
func (db *DB) DeletePost(id string) error {
	query := `DELETE FROM posts WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// GetAllProjects retrieves all projects from the database
func (db *DB) GetAllProjects() ([]models.Project, error) {
	query := `SELECT name, description, url, tech, status, icon FROM projects ORDER BY name`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.Name, &p.Description, &p.URL, &p.Tech, &p.Status, &p.Icon); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}

	return projects, rows.Err()
}

// SaveProject creates or updates a project
func (db *DB) SaveProject(project *models.Project) error {
	// Check if project exists by name
	var exists bool
	err := db.conn.QueryRow(`SELECT EXISTS(SELECT 1 FROM projects WHERE name = ?)`, project.Name).Scan(&exists)
	if err != nil {
		return err
	}

	if exists {
		query := `
		UPDATE projects 
		SET description = ?, url = ?, tech = ?, status = ?, icon = ?, updated_at = CURRENT_TIMESTAMP
		WHERE name = ?
		`
		_, err = db.conn.Exec(query, project.Description, project.URL, project.Tech, project.Status, project.Icon, project.Name)
	} else {
		query := `
		INSERT INTO projects (name, description, url, tech, status, icon)
		VALUES (?, ?, ?, ?, ?, ?)
		`
		_, err = db.conn.Exec(query, project.Name, project.Description, project.URL, project.Tech, project.Status, project.Icon)
	}

	return err
}

// MigrateFromYAML imports data from YAML files into the database
func (db *DB) MigrateFromYAML(posts []models.Post, projects []models.Project) error {
	log.Printf("Migrating %d posts and %d projects from YAML to database", len(posts), len(projects))

	// Migrate posts
	for _, post := range posts {
		if post.Date.IsZero() {
			post.Date = time.Now()
		}
		if err := db.SavePost(&post); err != nil {
			return fmt.Errorf("migrating post %s: %w", post.ID, err)
		}
	}

	// Migrate projects
	for _, project := range projects {
		if err := db.SaveProject(&project); err != nil {
			return fmt.Errorf("migrating project %s: %w", project.Name, err)
		}
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

// IncrementPostViews increments the view count for a post
func (db *DB) IncrementPostViews(postID string) error {
	query := `UPDATE posts SET views = views + 1 WHERE id = ?`
	_, err := db.conn.Exec(query, postID)
	return err
}

// GetPopularPosts retrieves posts ordered by view count
func (db *DB) GetPopularPosts(limit int) ([]models.Post, error) {
	query := `SELECT id, title, date, category, summary, content, tags, COALESCE(views, 0) 
	          FROM posts 
	          ORDER BY views DESC, date DESC 
	          LIMIT ?`
	rows, err := db.conn.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []models.Post
	for rows.Next() {
		var p models.Post
		var tags string
		if err := rows.Scan(&p.ID, &p.Title, &p.Date, &p.Category, &p.Summary, &p.Content, &tags, &p.Views); err != nil {
			return nil, err
		}
		if tags != "" {
			p.Tags = parseTagsFromString(tags)
		}
		posts = append(posts, p)
	}

	return posts, rows.Err()
}
