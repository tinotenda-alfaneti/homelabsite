package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/tinotenda-alfaneti/homelabsite/models"
	"gopkg.in/yaml.v3"
)

// GetEnv retrieves an environment variable or returns the default value
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// GetConfigPath returns the config file path, with smart defaults for different environments
func GetConfigPath() string {
	// Priority:
	// 1. CONFIG_PATH env var (Kubernetes sets this)
	// 2. Check if /app/config/config.yaml exists (Kubernetes PVC)
	// 3. Check if /srv/config/config.yaml exists (Kubernetes ConfigMap)
	// 4. Fall back to local path

	if path := os.Getenv("CONFIG_PATH"); path != "" {
		return path
	}

	// Check Kubernetes PVC path
	if _, err := os.Stat("/app/config/config.yaml"); err == nil {
		return "/app/config/config.yaml"
	}

	// Check Kubernetes ConfigMap path
	if _, err := os.Stat("/srv/config/config.yaml"); err == nil {
		return "/srv/config/config.yaml"
	}

	// Local development path
	return "config/config.yaml"
}

// GetDataDir returns the directory where data files are stored
func GetDataDir(configPath string) string {
	// If using PVC path (/app/config), data files are in /app/data
	if configPath == "/app/config/config.yaml" {
		return "/app/data"
	}

	// If using ConfigMap path, data files might be in PVC at /app/data
	if configPath == "/srv/config/config.yaml" {
		if _, err := os.Stat("/app/data"); err == nil {
			return "/app/data"
		}
	}

	// Local development path
	configDir := filepath.Dir(configPath)
	return filepath.Join(filepath.Dir(configDir), "data")
}

func Load(configPath string) (*models.Config, error) {
	// Load app configuration
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var appConfig models.AppConfig
	if err := yaml.Unmarshal(data, &appConfig); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Determine data directory
	dataDir := GetDataDir(configPath)

	// Load projects data
	projectsPath := filepath.Join(dataDir, "projects.yaml")
	projectsData, err := os.ReadFile(projectsPath)
	if err != nil {
		return nil, fmt.Errorf("reading projects: %w", err)
	}

	var projects models.ProjectsData
	if err := yaml.Unmarshal(projectsData, &projects); err != nil {
		return nil, fmt.Errorf("parsing projects: %w", err)
	}

	// Load posts data
	postsPath := filepath.Join(dataDir, "posts.yaml")
	postsData, err := os.ReadFile(postsPath)
	if err != nil {
		return nil, fmt.Errorf("reading posts: %w", err)
	}

	var posts models.PostsData
	if err := yaml.Unmarshal(postsData, &posts); err != nil {
		return nil, fmt.Errorf("parsing posts: %w", err)
	}

	// Sort posts by date descending
	sort.Slice(posts.Posts, func(i, j int) bool {
		return posts.Posts[i].Date.After(posts.Posts[j].Date)
	})

	return &models.Config{
		AppConfig: &appConfig,
		Projects:  projects.Projects,
		Posts:     posts.Posts,
	}, nil
}

// SaveData saves posts and projects data to their respective files
func SaveData(configPath string, cfg *models.Config) error {
	dataDir := GetDataDir(configPath)

	// Save projects
	projectsData := models.ProjectsData{Projects: cfg.Projects}
	data, err := yaml.Marshal(projectsData)
	if err != nil {
		return fmt.Errorf("marshaling projects: %w", err)
	}

	projectsPath := filepath.Join(dataDir, "projects.yaml")
	if err := os.WriteFile(projectsPath, data, 0600); err != nil {
		return fmt.Errorf("writing projects: %w", err)
	}

	// Save posts
	postsData := models.PostsData{Posts: cfg.Posts}
	data, err = yaml.Marshal(postsData)
	if err != nil {
		return fmt.Errorf("marshaling posts: %w", err)
	}

	postsPath := filepath.Join(dataDir, "posts.yaml")
	if err := os.WriteFile(postsPath, data, 0600); err != nil {
		return fmt.Errorf("writing posts: %w", err)
	}

	return nil
}

// Save is deprecated, use SaveData instead
// Kept for backwards compatibility
func Save(configPath string, cfg *models.Config) error {
	return SaveData(configPath, cfg)
}
