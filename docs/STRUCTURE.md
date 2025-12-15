# Project Structure

This Go application is now organized into clean, modular packages following best practices.

## Directory Structure

```
homelabsite/
├── main.go                 # Application entry point and routing setup
├── config/                 # Configuration management
│   └── config.go          # Load/save YAML config, environment variables
├── handlers/              # HTTP request handlers
│   ├── app.go            # App struct and shared utilities
│   ├── pages.go          # Page handlers (home, blog, services, etc.)
│   ├── auth.go           # Authentication handlers (login, logout)
│   └── api.go            # API endpoint handlers (posts, services CRUD)
├── middleware/            # HTTP middleware
│   └── auth.go           # Session-based authentication middleware
├── models/               # Data models
│   └── models.go         # Config, Service, Post structs
├── markdown/             # Markdown rendering
│   └── markdown.go       # Simple markdown to HTML converter
└── web/                  # Static assets and templates
    ├── static/
    │   ├── css/
    │   └── js/
    └── templates/

```

## Package Overview

### `main.go`
- Application bootstrap and configuration
- Router setup with all routes
- Embedded filesystem for static assets and templates
- Minimal code, delegates to packages

### `config/`
- Loads YAML configuration from file
- Saves configuration updates
- Environment variable helpers
- No business logic, pure config management

### `handlers/`
- **app.go**: Core App struct holding config, templates, auth, etc.
- **pages.go**: Handlers for rendering HTML pages
- **auth.go**: Login, logout, session management
- **api.go**: JSON API endpoints for CRUD operations
- All handlers are methods on the App struct

### `middleware/`
- **auth.go**: Session-based authentication
  - Session creation and validation
  - Automatic session cleanup
  - Thread-safe session storage

### `models/`
- Data structures used throughout the application
- Clean separation of data from logic
- Shared by config, handlers, and templates

### `markdown/`
- Simple markdown renderer
- Converts markdown strings to HTML
- Used via template function

## Benefits of This Structure

1. **Separation of Concerns**: Each package has a single, clear responsibility
2. **Testability**: Each package can be tested independently
3. **Maintainability**: Easy to locate and modify specific functionality
4. **Scalability**: Easy to add new handlers, middleware, or models
5. **Clean Dependencies**: Clear import paths, no circular dependencies

## Running the Application

```bash
# Development
go run .

# Build
go build -o homelabsite.exe .

# Run built binary
./homelabsite.exe
```

## Adding New Features

### New Page Handler
Add to `handlers/pages.go`:
```go
func (app *App) HandleNewPage(w http.ResponseWriter, r *http.Request) {
    data := map[string]interface{}{
        "Title": "New Page",
    }
    app.Render(w, "newpage.html", data)
}
```

### New API Endpoint
Add to `handlers/api.go`:
```go
func (app *App) HandleAPINewEndpoint(w http.ResponseWriter, r *http.Request) {
    // Your logic here
}
```

### New Middleware
Create new file in `middleware/` or add to existing files.

### New Model
Add to `models/models.go`:
```go
type NewModel struct {
    Field1 string `yaml:"field1" json:"field1"`
    Field2 int    `yaml:"field2" json:"field2"`
}
```
