package handlers

import (
	"net/http"
	"time"
)

func (app *App) HandleLoginPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"Title": "Admin Login - Atarnet Homelab",
		"Error": nil,
	}
	app.Render(w, "login.html", data)
}

func (app *App) HandleLogin(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	if !app.Auth.ValidateCredentials(username, password) {
		data := map[string]interface{}{
			"Title": "Admin Login - Atarnet Homelab",
			"Error": "Invalid username or password",
		}
		app.Render(w, "login.html", data)
		return
	}

	// Create session
	sessionToken, expiry := app.Auth.CreateSession()

	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    sessionToken,
		Expires:  expiry,
		HttpOnly: true,
		Path:     "/",
	})

	http.Redirect(w, r, "/admin", http.StatusFound)
}

func (app *App) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		app.Auth.DeleteSession(cookie.Value)
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
