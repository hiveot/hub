package login

import (
	"github.com/google/uuid"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/views/layouts"
	"html/template"
	"net/http"
	"time"
)

// GetLogin renders the login view
func GetLogin(t *template.Template, sm *session.SessionManager) http.HandlerFunc {

	// GET /login shows the login form
	return func(w http.ResponseWriter, r *http.Request) {
		data := map[string]any{
			// TODO: remember last login
			"loginID":  "",
			"password": "",
		}
		layouts.RenderWithLayout(w, t, "login.html", "main.html", data)
	}
}

// PostLogin handles the login request to log in with a password
func PostLogin(templates *template.Template, sm *session.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// obtain login form fields
		loginID := r.FormValue("loginID")
		password := r.FormValue("password")
		if loginID == "" && password == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		// invoke auth handler
		var err error = nil

		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		// store the
		//session, err := sm.Open(loginID, password)
		sessionID := uuid.NewString()
		expiresAt := time.Now().Add(3600 * time.Second)

		http.SetCookie(w, &http.Cookie{
			Name:    "session",
			Value:   sessionID,
			Expires: expiresAt,
			//Secure:   true,  // Cookie is only sent over HTTPS
			HttpOnly: true, // Cookie is not accessible via client-side java (CSRA attack)
		})
		//store session cookie
		// refresh app view and show dashboard

	}
}
