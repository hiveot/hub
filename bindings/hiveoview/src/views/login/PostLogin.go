package login

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"log/slog"
	"net/http"
)

// keep session auth for 7 days
// TODO: use the token expiry instead
//const DefaultAuthAge = 3600 * 24 * 7

// PostLogin handles the login request to log in with a password.
// This creates or refreshes a user session containing credentials.
// If connection fails then an error is returned.
func PostLogin(w http.ResponseWriter, r *http.Request) {
	sm := session.GetSessionManager()

	// obtain login form fields
	loginID := r.FormValue("loginID")
	password := r.FormValue("password")
	if loginID == "" && password == "" {
		http.Redirect(w, r, "/", http.StatusBadRequest)
		//w.WriteHeader(http.StatusBadRequest)
		return
	}

	// step 1: authenticate with the password
	hc, err := sm.ConnectWithPassword(loginID, password)
	if err != nil {
		slog.Warn("PostLogin failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("loginID", loginID),
			slog.String("err", err.Error()))
		// do not cache the login form in the browser
		w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		http.Redirect(w, r, "/login?error="+err.Error(), http.StatusSeeOther)
		return
	}
	// TODO: session age from config
	_, _ = sm.ActivateNewSession(w, r, hc)

	// update the session. This ensures an active session exists and the
	// cookie contains the existing or new session ID with a fresh auth token.
	// keep the session cookie for 30 days (todo: make this a service config)
	//maxAge := 3600 * 24 * 30
	//sm.LoginToSession(w, r, hc, maxAge)

	slog.Info("login successful", "loginID", loginID)
	// do not cache the password
	w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
	// prevent the browser from re-posting on back button or refresh (POST-Redirect-GET) pattern
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
