package session

import (
	"net/http"
)

// SessionLogout logs out of the current session
// This disconnects the session from the Hub and removes the auth cookie.
func SessionLogout(w http.ResponseWriter, r *http.Request) {
	sm := GetSessionManager()
	// in this case we need the cookie as the context might not be set
	_, claims, _ := sm.GetSessionFromCookie(r)
	if claims != nil {
		_ = sm.Close(claims.ID)
	}

	RemoveSessionCookie(w, r)

	// logout with a redirect
	//https://www.reddit.com/r/htmx/comments/s36zx2/how_do_you_use_hxredirect/
	isHtmx := r.Header.Get("HX-Request") != ""
	if isHtmx {
		w.Header().Add("hx-redirect", "/login")
		http.Redirect(w, r, "/login", http.StatusOK)
	} else {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}
