package httpstransport

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/tlsserver"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"io"
	"net/http"
)

// Experimental Digitwin REST handlers
// A convenience api to login, logout, read the directory and values of things

// HandleGetEvents returns a list of latest messages from a Thing
// Parameters: thingID
func (svc *HttpsTransport) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	cs, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message.
	args := outbox.ReadLatestArgs{ThingID: thingID}
	// just a single key supported
	if key != "" {
		args.Keys = []string{key}
	}
	argsJSON, _ := json.Marshal(args)
	msg := things.NewThingMessage(
		vocab.MessageTypeAction, outbox.ThingID, outbox.ReadLatestMethod, argsJSON, cs.GetClientID())
	stat := svc.handleMessage(msg)
	_ = stat
	resp := outbox.ReadLatestResp{}
	_ = json.Unmarshal(stat.Reply, &resp)
	// The response values are already serialized
	svc.writeReply(w, []byte(resp.Values), err)
}

// HandleGetThings returns a list of things in the directory
// No parameters
func (svc *HttpsTransport) HandleGetThings(w http.ResponseWriter, r *http.Request) {
}

func (svc *HttpsTransport) HandlePostLogout(w http.ResponseWriter, r *http.Request) {
	cs, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	// logout closes the session
	cs.Close()
	// TODO: remove client session cookie
	//svc.sessionManager.ClearSessionCookie(cs.sessionID)
}

// HandlePostLogin handles a login request and a new session, posted by a consumer
func (svc *HttpsTransport) HandlePostLogin(w http.ResponseWriter, r *http.Request) {
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

func (svc *HttpsTransport) HandlePostRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	args := api.RefreshTokenArgs{}
	var reply []byte
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
	if err == nil {
		resp := &api.RefreshTokenResp{Token: newToken}
		reply, err = json.Marshal(resp)
	}
	svc.writeReply(w, reply, err)
	// TODO: update client session cookie with new token
	//svc.sessionManager.SetSessionCookie(cs.sessionID,newToken)
}
