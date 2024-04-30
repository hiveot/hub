// Package rest with handling of rest API
package rest

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/protocols/httpsbinding/sessions"
	"github.com/hiveot/hub/runtime/router"
	"github.com/hiveot/hub/runtime/tlsserver"
	"io"
	"net/http"
)

// AuthnRest contains the digitwin REST API handlers for authentication
// with login/refresh tokens
type AuthnRest struct {
	RestHandler
	sessionAuth api.IAuthenticator
}

// HandlePostLogin handles a login request and a new session, posted by a consumer
func (svc *AuthnRest) HandlePostLogin(w http.ResponseWriter, r *http.Request) {
	sm := sessions.GetSessionManager()

	args := api.LoginArgs{}
	// credentials are in a json payload
	data, err := io.ReadAll(r.Body)
	if err == nil {
		err = json.Unmarshal(data, &args)
	}
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = json.Unmarshal(data, &args)
	// login generates a new session ID
	token, sid, err := svc.sessionAuth.Login(args.ClientID, args.Password, "")
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	// if a session exists, remove it
	oldToken, err := tlsserver.GetBearerToken(r)
	if err == nil {
		_, oldSid, err := svc.sessionAuth.ValidateToken(oldToken)
		if err == nil {
			_ = sm.Close(oldSid)
		}
	}
	// create the session for this token
	_, err = sm.NewSession(args.ClientID, r.RemoteAddr, sid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	reply := api.LoginResp{Token: token}
	resp, err := json.Marshal(reply)
	_, _ = w.Write(resp)
	w.WriteHeader(http.StatusOK)
	// TODO: set client session cookie
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
}

func (svc *AuthnRest) HandlePostRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	args := api.RefreshTokenArgs{}
	cs, _, _, data, err := svc.getRequestParams(r)
	if err == nil {
		err = json.Unmarshal(data, &args)
	}
	if cs.GetClientID() != args.ClientID {
		http.Error(w, "bad login", http.StatusUnauthorized)
		return
	}
	if err == nil {
		newToken, err = svc.sessionAuth.RefreshToken(cs.GetClientID(), args.OldToken, 0)
	}
	reply := &api.RefreshTokenResp{Token: newToken}
	resp, err := json.Marshal(reply)
	svc.writeReply(w, resp, err)
	// TODO: update client session cookie with new token
	//svc.sessionManager.SetSessionCookie(cs.sessionID,newToken)
}

func (svc *AuthnRest) HandlePostLogout(w http.ResponseWriter, r *http.Request) {
	cs, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	// logout closes the session
	cs.Close()
	// TODO: remove client session cookie
	//svc.sessionManager.ClearSessionCookie(cs.sessionID)
}

// RegisterMethods registers the authn rest methods.
// Note that HandleLogin should be registered separately outside the secured methods
func (svc *AuthnRest) RegisterMethods(r chi.Router) {
	// handlers for both agent and consumer messages
	// TODO: this should be handled by the session authenticator
	r.Post(vocab.PostRefreshPath, svc.HandlePostRefresh)
	r.Post(vocab.PostLogoutPath, svc.HandlePostLogout)
}

// NewAuthnRest creates a new instance of the Authn service REST API
func NewAuthnRest(handleMessage router.MessageHandler, sessionAuth api.IAuthenticator) *AuthnRest {
	svc := &AuthnRest{
		RestHandler: RestHandler{handleMessage: handleMessage},
		sessionAuth: sessionAuth,
	}
	return svc
}
