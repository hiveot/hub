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
		"loginID":  "",
		"password": "",
	}
	sessionContext := r.Context().Value("session")
	if sessionContext != nil {
		cs := sessionContext.(*session.ClientSession)
		data["loginID"] = cs.LoginID
	}
	// don't cache the login
	w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
	assets.RenderWithLayout(w, assets.AllTemplates, "login.html", "", data)
}

// PostLogin handles the login request to log in with a password.
// This creates or refreshes a session containing a hub connection for the user.
// If connection fails then the session persists but an error is returned.
func PostLogin(w http.ResponseWriter, r *http.Request) {
	var sessionID string

	// obtain login form fields
	loginID := r.FormValue("loginID")
	password := r.FormValue("password")
	if loginID == "" && password == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// if a session exists, reuse it
	sessionCookie, err := r.Cookie("session")
	if err == nil {
		// use the existing session cookie
		sessionID = sessionCookie.Value
	} else {
		sessionID = uuid.NewString()
		sessionCookie = &http.Cookie{
			Name:  "session",
			Value: sessionID,
			//Expires: see below
			//Secure:   true,  // Cookie is only sent over HTTPS
			HttpOnly: true, // Cookie is not accessible via client-side java (CSRA attack)
		}
	}
	sm := session.GetSessionManager()
	// add the session or return the existing one
	cs := sm.Add(sessionID, loginID, r.RemoteAddr, "")
	sessionCookie.Expires = cs.Expiry

	// attempt to connect the session using the given password
	// on success the session will obtain and store an auth token for reconnecting
	// without requiring a password.
	err = cs.ConnectWithPassword(password)
	if err != nil {
		slog.Warn("Login failed", "loginID", loginID, "err", err.Error())
		// do not cache the login form in the browser
		w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	// update the cookie with the new expiry
	sessionCookie.Expires = cs.Expiry
	http.SetCookie(w, sessionCookie)
	sm.Save() // save the session auth token

	slog.Info("login successful", "loginID", loginID, "sessionID", sessionID, "remoteAddr", r.RemoteAddr)
	// do not cache the password
	w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
	//TODO: return to last page?
	http.Redirect(w, r, "/", http.StatusFound)
}

// PostLogout removes the current session
func PostLogout(w http.ResponseWriter, r *http.Request) {
	cs := session.GetClientSession(r)
	if cs != nil {
		sm := session.GetSessionManager()
		sm.Remove(cs.SessionID)
	}
	// session is no longer valid
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    "logged out",
		MaxAge:   -1,
		Expires:  time.Now().Add(-time.Hour),
		HttpOnly: true, // Cookie is not accessible via client-side java (CSRA attack)
	})
	// redirect to root page
	http.Redirect(w, r, "/", http.StatusFound)
}
