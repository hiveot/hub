package session

import (
	"net/http"
)

// SessionLogout logs out of the current session
// This disconnects the session from the Hub and removes the auth cookie.
func SessionLogout(w http.ResponseWriter, r *http.Request) {
	cs, _ := GetSession(w, r)
	if cs != nil {
		sm := GetSessionManager()
		sm.Close(cs.sessionID)
	}

	RemoveSessionCookie(w, r)

	// redirect to home
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
