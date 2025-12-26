package handlers

import (
	"html/template"
	"log"
	"net/http"

	"github.com/tinotenda-alfaneti/homelabsite/cache"
	"github.com/tinotenda-alfaneti/homelabsite/db"
	"github.com/tinotenda-alfaneti/homelabsite/middleware"
	"github.com/tinotenda-alfaneti/homelabsite/models"
)

type App struct {
	Config     *models.Config
	Templates  *template.Template
	Auth       *middleware.AuthMiddleware
	ConfigPath string
	DB         *db.DB
	Cache      *cache.Cache
}

func (app *App) Render(w http.ResponseWriter, tmpl string, data map[string]interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := app.Templates.ExecuteTemplate(w, tmpl, data); err != nil {
		log.Printf("Error rendering template %s: %v", tmpl, err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
