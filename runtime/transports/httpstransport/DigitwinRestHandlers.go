package httpstransport

import (
	"encoding/json"
	"errors"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/tlsserver"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// Experimental Digitwin REST handlers
// A convenience api to login, logout, read the directory and values of things

// HandleGetEvents returns a list of latest messages from a Thing
// Parameters: thingID
func (svc *HttpsTransport) HandleGetEvents(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message.
	args := digitwin.OutboxReadLatestArgs{ThingID: thingID}
	// only a single key is supported at the moment
	if key != "" {
		args.Keys = []string{key}
	}
	//argsJSON, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction,
		digitwin.OutboxDThingID, digitwin.OutboxReadLatestMethod,
		args, cs.GetClientID())

	stat := svc.handleMessage(msg)
	//var reply []byte
	if stat.Error == "" {
		//var resp string
		// the serialized response contains a serialized message map.
		// this double serialization is because the outbox response is defined
		// as a serialized message map. Just reply with the once-serialized map
		//err, _ = stat.UnmarshalReply(&resp)
		// The response values are already serialized
		//reply = []byte(resp)
		err = nil
	} else {
		err = errors.New(stat.Error)
	}
	svc.writeReply(w, []byte(stat.Reply), err)
}

// HandleGetThings returns a list of things in the directory
// No parameters
func (svc *HttpsTransport) HandleGetThings(w http.ResponseWriter, r *http.Request) {
	cs, _, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message.
	limit := 100
	offset := 0
	if r.URL.Query().Has("limit") {
		limitStr := r.URL.Query().Get("limit")
		limit32, _ := strconv.ParseInt(limitStr, 10, 32)
		limit = int(limit32)
	}
	if r.URL.Query().Has("offset") {
		offsetStr := r.URL.Query().Get("offset")
		offset32, _ := strconv.ParseInt(offsetStr, 10, 32)
		offset = int(offset32)
	}
	args := digitwin.DirectoryReadTDsArgs{Limit: limit, Offset: offset}
	//argsJSON, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction,
		digitwin.DirectoryDThingID, digitwin.DirectoryReadTDsMethod,
		args, cs.GetClientID())

	stat := svc.handleMessage(msg)

	var reply []byte
	if stat.Error == "" {
		// the response is a serialized list of serialized TDs
		// just return as-is
		reply = []byte(stat.Reply)
		err = nil
	} else {
		err = errors.New(stat.Error)
	}
	svc.writeReply(w, reply, err)
}

// HandleGetThing returns the TD of a thing in the directory
// URL parameter {thingID}
func (svc *HttpsTransport) HandleGetThing(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// FIXME: are single text arguments serialized?
	// payload is always serialized ?
	msg := things.NewThingMessage(vocab.MessageTypeAction,
		digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod,
		thingID, cs.GetClientID())
	stat := svc.handleMessage(msg)

	var reply []byte
	if stat.Error == "" {
		// the response is a serialized TD
		// just return as-is
		reply = []byte(stat.Reply)
		err = nil
	} else {
		err = errors.New(stat.Error)
	}
	svc.writeReply(w, reply, err)
}

func (svc *HttpsTransport) HandlePostLogout(w http.ResponseWriter, r *http.Request) {
	cs, _, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	// logout closes the session
	cs.Close()
	// TODO: remove client session cookie
	//svc.sessionManager.ClearSessionCookie(cs.sessionID)
}

// HandlePostLogin handles a login request and a new session, posted by a consumer
// This uses the configured session authenticator.
func (svc *HttpsTransport) HandlePostLogin(w http.ResponseWriter, r *http.Request) {
	sm := sessions.GetSessionManager()

	args := authn.UserLoginArgs{}
	resp := authn.UserLoginResp{}
	// credentials are in a json payload
	data, err := io.ReadAll(r.Body)
	if err == nil {
		err = json.Unmarshal(data, &args)
	}
	if err != nil {
		slog.Warn("HandlePostLogin: parameter error", "err", err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	token, sid, err := svc.authenticator.Login(args.ClientID, args.Password)
	if err != nil {
		if err != nil {
			slog.Warn("HandlePostLogin: authentication error", "clientID", args.ClientID)
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
	}
	// remove existing session, if any
	oldToken, err := tlsserver.GetBearerToken(r)
	if err == nil {
		_, oldSid, err := svc.authenticator.ValidateToken(oldToken)
		if err == nil {
			_ = sm.Close(oldSid)
		}
	}
	// create the session for this token
	_, err = sm.NewSession(args.ClientID, r.RemoteAddr, sid)
	if err != nil {
		slog.Warn("HandlePostLogin: session error", "err", err.Error())
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	resp.SessionID = sid
	resp.Token = token
	respJson, err := json.Marshal(resp)
	// write sets statusOK
	_, _ = w.Write(respJson)
	slog.Info("HandlePostLogin: success", "clientID", args.ClientID)
	//w.WriteHeader(http.StatusOK)
	// TODO: set client session cookie
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
}

// HandlePostRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (svc *HttpsTransport) HandlePostRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string
	var resp []byte

	args := authn.UserRefreshTokenArgs{}
	cs, _, _, _, data, err := svc.getRequestParams(r)
	if err == nil {
		err = json.Unmarshal(data, &args)
	}
	// the session owner must match the token requested client ID
	if err != nil || cs.GetClientID() != args.ClientID {
		http.Error(w, "bad login", http.StatusUnauthorized)
		return
	}
	newToken, err = svc.authenticator.RefreshToken(args.ClientID, args.OldToken)
	if err == nil {
		resp, err = json.Marshal(newToken)
	}
	svc.writeReply(w, resp, err)
	// TODO: update client session cookie with new token
	//svc.sessionManager.SetSessionCookie(cs.sessionID,newToken)
}

// HandleSubscribe handles a subscription request
func (svc *HttpsTransport) HandleSubscribe(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		slog.Warn("HandleSubscribe", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleSubscribe",
		slog.String("clientID", cs.GetClientID()),
		slog.String("thingID", thingID),
		slog.String("key", key),
		slog.String("sessionID", cs.GetSessionID()),
		slog.Int("nr sse connections", cs.GetNrConnections()))
	cs.Subscribe(thingID, key)
}

// HandleUnsubscribe handles removal of a subscription request
func (svc *HttpsTransport) HandleUnsubscribe(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnsubscribe")
	cs, _, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cs.Unsubscribe(thingID, key)
}
