package httpstransport

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// Protocol handler implementation for support Form operations.

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
	svc.writeStatReply(w, stat)
}

// HandleGetThing returns the TD of a thing in the directory
// URL parameter {thingID}
func (svc *HttpsTransport) HandleGetThing(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	msg := things.NewThingMessage(vocab.MessageTypeAction,
		digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod,
		thingID, cs.GetClientID())
	stat := svc.handleMessage(msg)
	svc.writeStatReply(w, stat)
}

func (svc *HttpsTransport) HandlePostLogout(w http.ResponseWriter, r *http.Request) {
	cs, _, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	// logout closes the session which invalidates it
	cs.Close()
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
	// TODO: set client session cookie for browser clients
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
}

func (svc *HttpsTransport) HandlePostInvokeAction(w http.ResponseWriter, r *http.Request) {
	svc.handlePostMessage(vocab.MessageTypeAction, w, r)
}
func (svc *HttpsTransport) HandlePostPublishEvent(w http.ResponseWriter, r *http.Request) {
	svc.handlePostMessage(vocab.MessageTypeEvent, w, r)
}
func (svc *HttpsTransport) HandleWriteProperty(w http.ResponseWriter, r *http.Request) {
	svc.handlePostMessage(vocab.MessageTypeProperty, w, r)
}
func (svc *HttpsTransport) HandlePostTDD(w http.ResponseWriter, r *http.Request) {
	// TDD are posted as events with the $td key
	svc.handlePostMessage(vocab.MessageTypeEvent, w, r)
}

// handlePostMessage passes a posted action, event, property or TD request to the handler
// This unmarshals the payload and constructs a ThingMessage instance to pass to the handler.
// this contains optional query parameter for messageID
func (svc *HttpsTransport) handlePostMessage(messageType string, w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, key, body, err := svc.getRequestParams(r)
	var payload any

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	messageID := r.URL.Query().Get("messageID")
	if messageID == "" {
		messageID = uuid.NewString()
	}
	if body != nil && len(body) > 0 {
		err = json.Unmarshal(body, &payload)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	// turn the request into a Thing Message
	msg := things.NewThingMessage(
		messageType, thingID, key, payload, cs.GetClientID())
	msg.MessageID = messageID

	stat := svc.handleMessage(msg)
	reply, err := json.Marshal(&stat)
	svc.writeReply(w, reply, err)
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

// HandleReadAllEvents returns a list of latest event values from a Thing
// Parameters: thingID
func (svc *HttpsTransport) HandleReadAllEvents(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message.
	args := digitwin.OutboxReadLatestArgs{
		ThingID: thingID,
	}
	// only a single key is supported at the moment
	if key != "" {
		args.Keys = []string{key}
	}
	//argsJSON, _ := json.Marshal(args)
	msg := things.NewThingMessage(vocab.MessageTypeAction,
		digitwin.OutboxDThingID, digitwin.OutboxReadLatestMethod,
		args, cs.GetClientID())

	stat := svc.handleMessage(msg)
	svc.writeStatReply(w, stat)
}

// HandleReadAllProperties was added to the top level TD form. Handle it here.
func (svc *HttpsTransport) HandleReadAllProperties(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message to the outbox.
	args := digitwin.OutboxReadLatestArgs{
		MessageType: "events",
		ThingID:     thingID,
	}
	msg := things.NewThingMessage(vocab.MessageTypeAction,
		digitwin.OutboxDThingID, digitwin.OutboxReadLatestMethod,
		args, cs.GetClientID())

	stat := svc.handleMessage(msg)
	svc.writeStatReply(w, stat)
}

func (svc *HttpsTransport) HandleReadProperty(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message to the outbox.
	args := digitwin.OutboxReadLatestArgs{
		MessageType: "events",
		ThingID:     thingID,
		Keys:        []string{key},
	}
	msg := things.NewThingMessage(vocab.MessageTypeAction,
		digitwin.OutboxDThingID, digitwin.OutboxReadLatestMethod,
		args, cs.GetClientID())

	stat := svc.handleMessage(msg)
	svc.writeStatReply(w, stat)
}

// HandleSubscribeEvents handles a subscription request
func (svc *HttpsTransport) HandleSubscribeEvents(w http.ResponseWriter, r *http.Request) {
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

// HandleUnsubscribeEvents handles removal of a subscription request
func (svc *HttpsTransport) HandleUnsubscribeEvents(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnsubscribe")
	cs, _, thingID, key, _, err := svc.getRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cs.Unsubscribe(thingID, key)
}
