package session

import (
	"net/http"
)

// SessionLogout logs out of the current session
// This removes the cookie and logs out from the hub.
func SessionLogout(w http.ResponseWriter, r *http.Request) {
	_, sess, err := GetSessionFromContext(r)
	if err != nil {
		// this breaks redirect, so don't return a status code
		//http.Error(w, err.Error(), http.StatusUnauthorized)
	}
	if sess != nil {
		// logout will disconnect from the hub and remove the session.
		sess.Logout()
	}
	RemoveSessionCookie(w, r)

	// logout with a redirect
	//https://www.reddit.com/r/htmx/comments/s36zx2/how_do_you_use_hxredirect/
	isHtmx := r.Header.Get("HX-Request") != ""
	if isHtmx {
		w.Header().Add("hx-redirect", "/login")
		http.Redirect(w, r, "/login", http.StatusOK)
	} else {
		http.Redirect(w, r, "/login", http.StatusFound)
	}
}
