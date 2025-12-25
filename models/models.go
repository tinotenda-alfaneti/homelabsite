package models

import "time"

// AppConfig holds application-level configuration
type AppConfig struct {
	App struct {
		Name        string `yaml:"name"`
		Version     string `yaml:"version"`
		Environment string `yaml:"environment"`
	} `yaml:"app"`
	Server struct {
		Port         int    `yaml:"port"`
		ReadTimeout  string `yaml:"read_timeout"`
		WriteTimeout string `yaml:"write_timeout"`
		IdleTimeout  string `yaml:"idle_timeout"`
	} `yaml:"server"`
	Features struct {
		AdminEnabled    bool `yaml:"admin_enabled"`
		BlogEnabled     bool `yaml:"blog_enabled"`
		ServicesEnabled bool `yaml:"services_enabled"`
	} `yaml:"features"`
	Data struct {
		PostsFile    string `yaml:"posts_file"`
		ServicesFile string `yaml:"services_file"`
	} `yaml:"data"`
}

// ServicesData holds the services content
type ServicesData struct {
	Services []Service `yaml:"services"`
}

// PostsData holds the blog posts content
type PostsData struct {
	Posts []Post `yaml:"posts"`
}

// Config is the combined runtime configuration
type Config struct {
	AppConfig *AppConfig
	Services  []Service
	Posts     []Post
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

type User struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Comment struct {
	ID          int       `json:"id"`
	PostID      string    `json:"post_id"`
	ParentID    *int      `json:"parent_id,omitempty"`
	AuthorName  string    `json:"author_name"`
	AuthorEmail string    `json:"author_email"`
	Content     string    `json:"content"`
	CreatedAt   time.Time `json:"created_at"`
	Approved    bool      `json:"approved"`
	Replies     []Comment `json:"replies,omitempty"`
}
