// Package httptransport with handlers for the http protocol
package httptransport

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/tlsserver"
	"github.com/hiveot/hub/runtime/transports/httptransport/subprotocols"
	jsoniter "github.com/json-iterator/go"
	"io"
	"log/slog"
	"net/http"
	"strconv"
)

// Http binding with form handler methods

// HandleLogout removes client session
func (svc *HttpBinding) HandleLogout(w http.ResponseWriter, r *http.Request) {

	sessID, clientID, err := subprotocols.GetSessionIdFromContext(r)
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		err := fmt.Errorf("Missing session for request '%s' from '%s'",
			r.RequestURI, r.RemoteAddr)
		slog.Error(err.Error())
		svc.writeError(w, err, http.StatusInternalServerError)
		return
	}
	slog.Info("HandleLogout", slog.String("clientID", clientID))
	_ = svc.cm.CloseClientConnections(sessID)
}

// HandleLogin handles a login request and a new session, posted by a consumer
// This uses the configured session authenticator.
func (svc *HttpBinding) HandleLogin(w http.ResponseWriter, r *http.Request) {

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
			sess, _ := svc.sm.GetSession(oldSid)
			if sess != nil {
				clientID := sess.GetClientID()
				svc.cm.CloseSessionConnections(clientID, oldSid)
			}
		}
	}
	// create the session for this token
	_, err = svc.sm.AddSession(args.ClientID, r.RemoteAddr, sid)
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

// HandleProgressUpdate sends a delivery update message to the digital twin
func (svc *HttpBinding) HandleProgressUpdate(w http.ResponseWriter, r *http.Request) {
	slog.Debug("HandleProgressUpdate")
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	stat := hubclient.DeliveryStatus{}
	err = jsoniter.Unmarshal(rp.Body, &stat)
	if err == nil {
		err = svc.hubRouter.HandleProgressUpdate(rp.ClientID, stat)
	}
	if err != nil {
		svc.writeError(w, err, http.StatusBadRequest)
	}
}

// HandleActionRequest requests an action from the digital twin
// NOTE: This returns a header with a dataschema if a schema from
// additionalResponses is returned.
//
// The sender must include the connection-id header of the connection it wants to
// receive the response.
func (svc *HttpBinding) HandleActionRequest(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	var input any

	slog.Debug("HandleActionRequest", slog.String("ClientID", rp.ClientID))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if rp.Body != nil && len(rp.Body) > 0 {
		err = json.Unmarshal(rp.Body, &input)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	// The client can provide a messageID for actions. Useful for associating
	// RPC type actions with a response.
	reqID := r.Header.Get(tlsclient.HTTPMessageIDHeader)

	// prefix the connectionID with the sessionID to prevent connection hijacking
	// this is matched with the cid created when the sub-protocol binding connected.
	cid := rp.SessionID + "-" + rp.ConnID

	status, output, messageID, err := svc.hubRouter.HandleActionFlow(
		rp.ThingID, rp.Name, input, reqID, rp.ClientID, cid)

	// there are 3 possible results:
	// on status completed; return output
	// on status failed: return http ok with DeliveryStatus containing error
	// on status other: return  DeliveryStatus object with progress
	//
	// This means that the result is either out or a DeliveryStatus object
	// Forms will have added:
	// ```
	//  "additionalResponses": [{
	//                    "success": false,
	//                    "contentType": "application/json",
	//                    "schema": "DeliveryStatus"
	//                }]
	//```
	replyHeader := w.Header()
	if replyHeader == nil {
		// this happened a few times during testing. perhaps a broken connection while debugging?
		err = fmt.Errorf("HandleActionRequest: Can't return result."+
			" Write header is nil. This is unexpected. clientID='%s", rp.ClientID)
		svc.writeError(w, err, http.StatusInternalServerError)
		return
	}
	if messageID != "" {
		replyHeader.Set(hubclient.MessageIDHeader, messageID)
	}

	// in case of error include the return data schema
	if err != nil {
		replyHeader.Set(hubclient.DataSchemaHeader, "DeliveryStatus")
		resp := hubclient.DeliveryStatus{
			MessageID: messageID,
			Progress:  status,
			Error:     err.Error(),
			Reply:     output,
		}
		svc.writeReply(w, resp)
		return
	} else if status != vocab.ProgressStatusCompleted {
		// if progress isn't completed then also return the delivery progress
		replyHeader.Set(hubclient.DataSchemaHeader, "DeliveryStatus")
		resp := hubclient.DeliveryStatus{
			MessageID: messageID,
			Progress:  status,
			Reply:     output,
		}
		svc.writeReply(w, resp)
		return
	}
	// TODO: standardize headers
	replyHeader.Set(hubclient.StatusHeader, status)

	// request completed, write output
	if output != nil {
		svc.writeReply(w, output)
	} else {
		svc.writeReply(w, nil)
	}
	return
}

// HandlePublishEvent update digitwin with event published by agent
func (svc *HttpBinding) HandlePublishEvent(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	var evValue any

	slog.Debug("HandlePublishEvent", slog.String("clientID", rp.ClientID))
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if rp.Body != nil && len(rp.Body) > 0 {
		err = jsoniter.Unmarshal(rp.Body, &evValue)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	// pass the event to the digitwin service for further processing
	messageID := r.Header.Get(hubclient.MessageIDHeader)
	err = svc.hubRouter.HandleEventFlow(rp.ClientID, rp.ThingID, rp.Name, evValue, messageID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, nil)
}

// HandleQueryAction returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleQueryAction(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	slog.Debug("HandleQueryAction", slog.String("ClientID", rp.ClientID))
	evList, err := svc.dtwService.ValuesSvc.QueryAction(rp.ClientID,
		digitwin.ValuesQueryActionArgs{ThingID: rp.ThingID, Name: rp.Name})
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, evList)
}

// HandleQueryAllActions returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleQueryAllActions(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	slog.Debug("HandleQueryAllActions", slog.String("ClientID", rp.ClientID))
	actList, err := svc.dtwService.ValuesSvc.QueryAllActions(rp.ClientID, rp.ThingID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, actList)
}

// HandleReadAllEvents returns a list of latest event values from a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleReadAllEvents(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	slog.Debug("HandleReadAllEvents", slog.String("ClientID", rp.ClientID))
	evList, err := svc.dtwService.ValuesSvc.ReadAllEvents(rp.ClientID, rp.ThingID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, evList)
}

// HandleReadAllProperties was added to the top level TD form. Handle it here.
func (svc *HttpBinding) HandleReadAllProperties(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	slog.Debug("HandleReadAllProperties", slog.String("ClientID", rp.ClientID))
	thing, err := svc.dtwService.ValuesSvc.ReadAllProperties(rp.ClientID, rp.ThingID)
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
func (svc *HttpBinding) HandleReadAllThings(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	slog.Debug("HandleReadAllThings", slog.String("ClientID", rp.ClientID))
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
	thingsList, err := svc.dtwService.DirSvc.ReadAllDTDs(rp.ClientID,
		digitwin.DirectoryReadAllDTDsArgs{Offset: offset, Limit: limit})
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, thingsList)
}

// HandleReadEvent returns the latest event value from a Thing
// Parameters: {thingID}, {name}
func (svc *HttpBinding) HandleReadEvent(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	slog.Debug("HandleReadEvent", slog.String("ClientID", rp.ClientID))
	evList, err := svc.dtwService.ValuesSvc.ReadEvent(rp.ClientID,
		digitwin.ValuesReadEventArgs{ThingID: rp.ThingID, Name: rp.Name})
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, evList)
}

func (svc *HttpBinding) HandleReadProperty(w http.ResponseWriter, r *http.Request) {
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	slog.Debug("HandleReadProperty", slog.String("clientID", rp.ClientID))
	thing, err := svc.dtwService.ValuesSvc.ReadProperty(rp.ClientID,
		digitwin.ValuesReadPropertyArgs{ThingID: rp.ThingID, Name: rp.Name})
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, &thing)
}

// HandleReadThing returns the TD of a thing in the directory
// URL parameter {thingID}
func (svc *HttpBinding) HandleReadThing(w http.ResponseWriter, r *http.Request) {

	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}

	slog.Debug("HandleReadThing", slog.String("clientID", rp.ClientID))
	thing, err := svc.dtwService.DirSvc.ReadDTD(rp.ClientID, rp.ThingID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, &thing)
}

// HandleRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (svc *HttpBinding) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	var newToken string

	args := authn.UserRefreshTokenArgs{}
	rp, err := subprotocols.GetRequestParams(r)
	if err == nil {
		err = json.Unmarshal(rp.Body, &args)
	}
	slog.Debug("HandleRefresh", slog.String("clientID", args.ClientID))
	// the session owner must match the token requested client ID
	if err != nil || rp.ClientID != args.ClientID {
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

// HandleUpdateThing agent sends a new TD document
func (svc *HttpBinding) HandleUpdateThing(w http.ResponseWriter, r *http.Request) {
	slog.Debug("HandleUpdateThing")
	rp, err := subprotocols.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tdJSON := ""
	err = jsoniter.Unmarshal(rp.Body, &tdJSON)
	err = svc.hubRouter.HandleUpdateTDFlow(rp.ClientID, tdJSON)
	if err != nil {
		svc.writeError(w, err, http.StatusBadRequest)
	}
}

// HandleUpdateProperty agent sends single or multiple property updates
func (svc *HttpBinding) HandleUpdateProperty(w http.ResponseWriter, r *http.Request) {

	rp, err := subprotocols.GetRequestParams(r)
	var value any

	slog.Debug("HandleUpdateProperty", slog.String("clientID", rp.ClientID))

	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if rp.Body != nil && len(rp.Body) > 0 {
		err = json.Unmarshal(rp.Body, &value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	}
	messageID := r.Header.Get(tlsclient.HTTPMessageIDHeader)
	err = svc.hubRouter.HandleUpdatePropertyFlow(rp.ClientID, rp.ThingID, rp.Name, value, messageID)
	if err != nil {
		svc.writeError(w, err, 0)
		return
	}
	svc.writeReply(w, nil)
}

// HandleWriteProperty consumer requests to update a Thing property
func (svc *HttpBinding) HandleWriteProperty(w http.ResponseWriter, r *http.Request) {

	rp, err := subprotocols.GetRequestParams(r)
	slog.Debug("HandleWriteProperty",
		slog.String("consumerID", rp.ClientID),
		slog.String("dThingID", rp.ThingID), slog.String("name", rp.Name))

	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	var newValue any
	var messageID string
	err = json.Unmarshal(rp.Body, &newValue)
	if err == nil {
		_, messageID, err = svc.hubRouter.HandleWritePropertyFlow(rp.ThingID, rp.Name, newValue, rp.ClientID)
	}
	if err != nil {
		svc.writeError(w, err, 0)
		return
	} else {
		w.Header().Set(tlsclient.HTTPMessageIDHeader, messageID)
	}
	svc.writeReply(w, nil)
}
