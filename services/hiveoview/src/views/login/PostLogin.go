package login

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"log/slog"
	"net/http"
)

// keep session auth for 7 days
// TODO: use the token expiry instead
//const DefaultAuthAge = 3600 * 24 * 7

// PostLoginHandler returns the handler for the password based login request.
// The handler creates or refreshes a user session containing credentials.
// If connection fails then an error is returned.
func PostLoginHandler(sm *session.WebSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// obtain login form fields
		loginID := r.FormValue("loginID")
		password := r.FormValue("password")
		if loginID == "" && password == "" {
			http.Redirect(w, r, src.RenderLoginPath, http.StatusBadRequest)
			//w.WriteHeader(http.StatusBadRequest)
			return
		}
		cid := r.Header.Get(hubclient.ConnectionIDHeader)
		slog.Info("PostLoginHandler",
			"loginID", loginID,
			"cid", cid)
		// step 1: authenticate with the password
		// FIXME: this connection becomes orphaned somehow as the web client
		// does not have a way to disconnect it. We just need a token.
		cs, err := sm.ConnectWithPassword(w, r, loginID, password, cid)
		_ = cs
		if err != nil {
			slog.Warn("PostLogin failed",
				slog.String("remoteAddr", r.RemoteAddr),
				slog.String("loginID", loginID),
				slog.String("err", err.Error()))
			// do not cache the login form in the browser
			w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
			http.Redirect(w, r, src.RenderLoginPath+"?error="+err.Error(), http.StatusSeeOther)
			return
		}

		// update the session. This ensures an active session exists and the
		// cookie contains the existing or new session ID with a fresh auth token.
		// keep the session cookie for 30 days (todo: make this a service config)
		//maxAge := 3600 * 24 * 30
		//sm.LoginToSession(w, r, hc, maxAge)

		slog.Info("login successful", "loginID", loginID)
		// do not cache the password
		w.Header().Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		// prevent the browser from re-posting on back button or refresh (POST-Redirect-GET) pattern
		http.Redirect(w, r, src.RenderDashboardRootPath, http.StatusSeeOther)
	}
}
