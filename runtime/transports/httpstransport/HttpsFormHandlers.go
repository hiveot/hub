package httpstransport

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// Protocol handler implementation for supported TD Form operations.
func (svc *HttpsTransport) HandleAgentPublishEvent(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, name, body, err := svc.getRequestParams(r)
	var evValue any
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if body != nil && len(body) > 0 {
		err = json.Unmarshal(body, &evValue)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}

	err = svc.dtwService.AddEventValue(cs.GetClientID(), thingID, name, evValue)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, nil)
}

func (svc *HttpsTransport) HandleLogout(w http.ResponseWriter, r *http.Request) {
	cs, _, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		return
	}
	// logout closes the session which invalidates it
	cs.Close()
}

// HandleLogin handles a login request and a new session, posted by a consumer
// This uses the configured session authenticator.
func (svc *HttpsTransport) HandleLogin(w http.ResponseWriter, r *http.Request) {
	sm := sessions.GetSessionManager()

	args := authn.UserLoginArgs{}
	resp := authn.UserLoginResp{}
	// credentials are in a json payload
	data, err := io.ReadAll(r.Body)
	if err == nil {
		err = json.Unmarshal(data, &args)
	}
	if err != nil {
		slog.Warn("HandleLogin: parameter error", "err", err.Error())
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}
	token, sid, err := svc.authenticator.Login(args.ClientID, args.Password)
	if err != nil {
		if err != nil {
			slog.Warn("HandleLogin: authentication error", "clientID", args.ClientID)
			svc.writeError(w, err, http.StatusUnauthorized)
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
		slog.Warn("HandleLogin: session error", "err", err.Error())
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}
	resp.SessionID = sid
	resp.Token = token
	slog.Info("HandleLogin: success", "clientID", args.ClientID)
	// TODO: set client session cookie for browser clients
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
	svc.writeReply(w, &resp)
}

func (svc *HttpsTransport) HandleInvokeAction(w http.ResponseWriter, r *http.Request) {
	cs, _, dThingID, name, body, err := svc.getRequestParams(r)
	var input any

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	//	messageID := r.URL.Query().Get("messageID")
	//
	if body != nil && len(body) > 0 {
		err = json.Unmarshal(body, &input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	output, status, err := svc.dtwService.InvokeAction(cs.GetClientID(), dThingID, name, input)
	_ = status
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	// TODO: Add a way to track progress, prolly through properties
	svc.writeReply(w, output)
}

// HandleQueryAction returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpsTransport) HandleQueryAllActions(w http.ResponseWriter, r *http.Request) {
	cs, _, dThingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	actList, err := svc.dtwService.ReadAllActions(cs.GetClientID(), dThingID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, actList)
}

// HandleQueryAction returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpsTransport) HandleQueryAction(w http.ResponseWriter, r *http.Request) {
	cs, _, dThingID, name, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	evList, err := svc.dtwService.ReadAction(cs.GetClientID(), dThingID, name)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, evList)
}

// HandleReadAllEvents returns a list of latest event values from a Thing
// Parameters: thingID
func (svc *HttpsTransport) HandleReadAllEvents(w http.ResponseWriter, r *http.Request) {
	cs, _, dThingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	evList, err := svc.dtwService.ReadAllEvents(cs.GetClientID(), dThingID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, evList)
}

// HandleReadAllProperties was added to the top level TD form. Handle it here.
func (svc *HttpsTransport) HandleReadAllProperties(w http.ResponseWriter, r *http.Request) {
	cs, _, dThingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	thing, err := svc.dtwService.ReadAllProperties(cs.GetClientID(), dThingID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, &thing)
}

// HandleReadAllThings returns a list of things in the directory
//
// Query params for paging:
//
//	limit=N, limit the number of results to N TD documents
//	offset=N, skip the first N results in the result
func (svc *HttpsTransport) HandleReadAllThings(w http.ResponseWriter, r *http.Request) {
	cs, _, _, _, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// this request can simply be turned into an action message for the directory.
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
	thingsList, err := svc.dtwService.ReadAllThings(cs.GetClientID(), offset, limit)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, thingsList)
}

// HandleReadEvent returns the latest event value from a Thing
// Parameters: {thingID}, {name}
func (svc *HttpsTransport) HandleReadEvent(w http.ResponseWriter, r *http.Request) {
	cs, _, dThingID, name, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	evList, err := svc.dtwService.ReadEvent(cs.GetClientID(), dThingID, name)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, evList)
}

func (svc *HttpsTransport) HandleReadProperty(w http.ResponseWriter, r *http.Request) {
	cs, _, dThingID, name, _, err := svc.getRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	thing, err := svc.dtwService.ReadProperty(cs.GetClientID(), dThingID, name)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, &thing)
}

// HandleReadThing returns the TD of a thing in the directory
// URL parameter {thingID}
func (svc *HttpsTransport) HandleReadThing(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}
	thing, err := svc.dtwService.ReadThing(cs.GetClientID(), thingID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, &thing)
}

// HandleRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (svc *HttpsTransport) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string

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
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, newToken)
	// TODO: update client session cookie with new token
	//svc.sessionManager.SetSessionCookie(cs.sessionID,newToken)
}

// HandleObserveProperty handles a property observe request for one or all properties
// FIXME: this is part of the sse-cs sub-protocol
func (svc *HttpsTransport) HandleObserveProperty(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, name, _, err := svc.getRequestParams(r)
	if err != nil {
		slog.Warn("HandleObserve", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleObserve",
		slog.String("clientID", cs.GetClientID()),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("sessionID", cs.GetSessionID()),
		slog.Int("nr sse connections", cs.GetNrConnections()))
	cs.ObserveProperty(thingID, name)
}

// HandleSubscribeEvent handles a subscription request for one or all events
// FIXME: this is part of the sse-cs sub-protocol
func (svc *HttpsTransport) HandleSubscribeEvent(w http.ResponseWriter, r *http.Request) {
	cs, _, thingID, name, _, err := svc.getRequestParams(r)
	if err != nil {
		slog.Warn("HandleSubscribe", "err", err.Error())
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	slog.Info("HandleSubscribe",
		slog.String("clientID", cs.GetClientID()),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.String("sessionID", cs.GetSessionID()),
		slog.Int("nr sse connections", cs.GetNrConnections()))
	cs.SubscribeEvent(thingID, name)
}

// HandleUnsubscribeEvent handles removal of one or all event subscriptions
// FIXME: this is part of the sse-cs sub-protocol
func (svc *HttpsTransport) HandleUnsubscribeEvent(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnsubscribe")
	cs, _, thingID, name, _, err := svc.getRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cs.UnsubscribeEvent(thingID, name)
}

// HandleUnobserveAllProperties handles removal of all property observe subscriptions
// FIXME: this is part of the sse-cs sub-protocol
func (svc *HttpsTransport) HandleUnobserveAllProperties(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnobserveAllProperties")
	cs, _, thingID, _, _, err := svc.getRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cs.UnobserveAllProperties(thingID)
}

// HandleUnobserveProperty handles removal of one property observe subscriptions
// FIXME: this is part of the sse-cs sub-protocol
func (svc *HttpsTransport) HandleUnobserveProperty(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUnobserveProperty")
	cs, _, thingID, name, _, err := svc.getRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	cs.UnobserveProperty(thingID, name)
}

// HandleUpdateThing agent sends a new TD document
func (svc *HttpsTransport) HandleUpdateThing(w http.ResponseWriter, r *http.Request) {
	slog.Info("HandleUpdateThing")
	cs, _, thingID, _, body, err := svc.getRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	svc.dtwService.UpdateTD(cs.GetClientID(), thingID, string(body))
}

// HandleWriteProperty handles the request to update a Thing property
func (svc *HttpsTransport) HandleWriteProperty(w http.ResponseWriter, r *http.Request) {

	cs, _, dThingID, name, body, err := svc.getRequestParams(r)
	slog.Info("HandleWriteProperty",
		"consumerID", cs.GetClientID(), "dThingID", dThingID, "name", name)

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var newValue any
	err = json.Unmarshal(body, &newValue)
	if err == nil {
		_, err = svc.dtwService.WriteProperty(cs.GetClientID(), dThingID, name, newValue)
	}
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, nil)
}
