package login

import (
	"encoding/json"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/transports/servers/httpserver"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log/slog"
	"net/http"
)

// keep session auth for 7 days
// TODO: use the token expiry instead
//const DefaultAuthAge = 3600 * 24 * 7

// PostLoginFormHandler returns the handler for the password based login request.
// The handler creates or refreshes a user session containing credentials.
// If connection fails then an error is returned.
//
// This requires a transports.ConnectionIDHeader (connection-id header)
// for a session to be retained.
func PostLoginFormHandler(sm *session.WebSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// obtain login form fields

		loginID := r.FormValue("loginID")
		password := r.FormValue("password")
		if loginID == "" && password == "" {
			http.Redirect(w, r, src.RenderLoginPath, http.StatusBadRequest)
			//w.WriteHeader(http.StatusBadRequest)
			return
		}
		cid := r.Header.Get(httpserver.ConnectionIDHeader)
		slog.Info("PostLoginFormHandler",
			"loginID", loginID,
			"cid", cid)

		newToken, err := sm.ConnectWithPassword(w, r, loginID, password, cid)
		_ = newToken
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
		header := w.Header()
		header.Add("Cache-Control", "no-cache, max-age=0, must-revalidate, no-store")
		// prevent the browser from re-posting on back button or refresh (POST-Redirect-GET) pattern
		http.Redirect(w, r, src.RenderDashboardRootPath, http.StatusSeeOther)
	}
}

// PostLoginHandler lets a client login using a password and returns a token.
//
// This requires a transports.ConnectionIDHeader (connection-id header)
// for a session to be retained.
// This returns a new authentication token that can be used as bearer token instead
// of logging in again.
func PostLoginHandler(sm *session.WebSessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return
		}
		loginMessage := map[string]string{}
		err = json.Unmarshal(body, &loginMessage)
		//	"login":    cl.GetClientID(),
		//	"password": password,
		//}
		// FIXME: use a shared login message struct
		loginID := loginMessage["login"]
		password := loginMessage["password"]
		if loginID == "" && password == "" {
			http.Redirect(w, r, src.RenderLoginPath, http.StatusBadRequest)
			//w.WriteHeader(http.StatusBadRequest)
			return
		}
		cid := r.Header.Get(httpserver.ConnectionIDHeader)
		slog.Info("PostLoginHandler",
			"loginID", loginID,
			"cid", cid)

		newToken, err := sm.ConnectWithPassword(w, r, loginID, password, cid)

		// this will prevent a redirect from working
		newTokenJSON, _ := jsoniter.Marshal(newToken)
		w.Write(newTokenJSON)

	}
}
