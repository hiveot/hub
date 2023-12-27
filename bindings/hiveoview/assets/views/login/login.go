package login

import (
	"github.com/google/uuid"
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
	"time"
)

// RenderLogin renders the login view
// TODO: Proper login form fragment
func RenderLogin(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		// TODO: remember last login
		"loginID":  "",
		"password": "",
	}
	assets.RenderWithLayout(w, assets.AllTemplates, "login.html", "", data)
}

// PostLogin handles the login request to log in with a password
func PostLogin(w http.ResponseWriter, r *http.Request) {
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

	//session, err := sm.Open(loginID, password)
	sessionID := uuid.NewString()
	expiresAt := time.Now().Add(3600 * time.Second)
	sm := session.GetSessionManager()
	si, err := sm.Add(sessionID, loginID, expiresAt, r.RemoteAddr)
	if err != nil {
		slog.Warn("Adding session failed", "loginID", loginID, "err", err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	//
	http.SetCookie(w, &http.Cookie{
		Name:    "session",
		Value:   sessionID,
		Expires: expiresAt,
		//Secure:   true,  // Cookie is only sent over HTTPS
		HttpOnly: true, // Cookie is not accessible via client-side java (CSRA attack)
	})

	err = si.ConnectWithPassword(password)
	if err != nil {
		slog.Warn("Login failed", "loginID", loginID, "err", err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(err.Error()))
		return
	}

	slog.Info("login successful", "loginID", loginID, "sessionID", sessionID, "remoteAddr", r.RemoteAddr)
	//TODO: return to last page?
	http.Redirect(w, r, "/", http.StatusFound)
}

// PostLogout removes the current session
func PostLogout(w http.ResponseWriter, r *http.Request) {
	// TODO: add support for logout
}
