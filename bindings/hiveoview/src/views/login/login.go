package login

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"net/http"
)

// RenderLogin renders the login form
func RenderLogin(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		"loginID": "",
	}
	loginError := r.URL.Query().Get("error")
	if loginError != "" {
		data["error"] = loginError
	}

	// don't cache the login
	// FIXME: delete the post from history so that a back button press doesn't re-post login cred.
	// apparently the cache control doesn't help for this.
	w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
	views.TM.RenderFull(w, "login.gohtml", data)
}
