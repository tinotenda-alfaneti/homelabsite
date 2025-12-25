package db

import (
	"strings"

	"github.com/tinotenda-alfaneti/homelabsite/models"
)

// SearchPosts performs full-text search on posts
func (db *DB) SearchPosts(query string) ([]models.Post, error) {
	if query == "" {
		return []models.Post{}, nil
	}

	// Search in title, content, tags, and category
	searchQuery := `
		SELECT id, title, date, category, summary, content, tags 
		FROM posts 
		WHERE 
			title LIKE ? OR 
			content LIKE ? OR 
			category LIKE ? OR 
			tags LIKE ?
		ORDER BY date DESC
	`

	searchPattern := "%" + query + "%"
	rows, err := db.conn.Query(searchQuery, searchPattern, searchPattern, searchPattern, searchPattern)
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
		if tags != "" {
			p.Tags = parseTagsFromString(tags)
		}
		posts = append(posts, p)
	}

	return posts, rows.Err()
}

// SearchPostsByTag finds posts that have a specific tag
func (db *DB) SearchPostsByTag(tag string) ([]models.Post, error) {
	if tag == "" {
		return []models.Post{}, nil
	}

	// Since tags are stored as comma-separated, we need to use LIKE
	searchQuery := `
		SELECT id, title, date, category, summary, content, tags 
		FROM posts 
		WHERE tags LIKE ?
		ORDER BY date DESC
	`

	// Match tag exactly or as part of comma-separated list
	patterns := []string{
		tag,               // exact match
		tag + ",%",        // at start
		"%," + tag,        // at end
		"%," + tag + ",%", // in middle
	}

	postsMap := make(map[string]models.Post)

	for _, pattern := range patterns {
		rows, err := db.conn.Query(searchQuery, "%"+pattern+"%")
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var p models.Post
			var tags string
			if err := rows.Scan(&p.ID, &p.Title, &p.Date, &p.Category, &p.Summary, &p.Content, &tags); err != nil {
				rows.Close()
				return nil, err
			}
			if tags != "" {
				p.Tags = parseTagsFromString(tags)
			}

			// Check if this tag actually exists in the parsed tags
			hasTag := false
			for _, t := range p.Tags {
				if strings.EqualFold(t, tag) {
					hasTag = true
					break
				}
			}

			if hasTag {
				postsMap[p.ID] = p
			}
		}
		rows.Close()
	}

	// Convert map to slice
	posts := make([]models.Post, 0, len(postsMap))
	for _, post := range postsMap {
		posts = append(posts, post)
	}

	return posts, nil
}

// GetAllTags returns all unique tags from all posts
func (db *DB) GetAllTags() ([]string, error) {
	query := `SELECT DISTINCT tags FROM posts WHERE tags != ''`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tagsMap := make(map[string]bool)
	for rows.Next() {
		var tagsStr string
		if err := rows.Scan(&tagsStr); err != nil {
			return nil, err
		}

		tags := parseTagsFromString(tagsStr)
		for _, tag := range tags {
			tagsMap[tag] = true
		}
	}

	// Convert map to slice
	allTags := make([]string, 0, len(tagsMap))
	for tag := range tagsMap {
		allTags = append(allTags, tag)
	}

	return allTags, rows.Err()
}
