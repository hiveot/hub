// Package httptransport with handlers for the http protocol
package httptransport

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/transports/httptransport/httpcontext"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
)

// Http binding with form handler methods

// HandleInvokeRequestProgress sends an action progress update message to the digital twin
func (svc *HttpBinding) HandleInvokeRequestProgress(w http.ResponseWriter, r *http.Request) {
	slog.Debug("HandlePublishRequestProgress")
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err == nil {
		var progressMsg hubclient.RequestProgress
		err = utils.Decode(rp.Data, &progressMsg)
		if err == nil {
			err = svc.digitwinRouter.HandlePublishRequestProgress(rp.ClientID, progressMsg)
		}
	}
	if err != nil {
		svc.writeError(w, err, http.StatusBadRequest)
	}
}

// HandleInvokeAction requests an action from the digital twin
// NOTE: This returns a header with a dataschema if a schema from
// additionalResponses is returned.
//
// The sender must include the connection-id header of the connection it wants to
// receive the response.
func (svc *HttpBinding) HandleInvokeAction(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)

	slog.Info("HandleInvokeAction",
		slog.String("SenderID", rp.ClientID),
		slog.String("clcid", rp.CLCID),
		slog.String("dThingID", rp.ThingID),
		slog.String("name", rp.Name),
		slog.String("RemoteAddr", r.RemoteAddr),
		slog.String("requestID", rp.RequestID),
	)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// an action request should have a cid when used with SSE.
	// for now just warn if its missing
	if r.Header.Get(hubclient.ConnectionIDHeader) == "" {
		slog.Warn("InvokeAction request without 'cid' header. This is needed for the SSE subprotocol",
			"clientID", rp.ClientID)
	}

	// optionally provide a requestID. one is generated and returned if not provided
	// really only needed for rpc's as the requestID must be known before
	// this returns to avoid a race condition.
	// TODO: maybe prefix this with the clientID to avoid hijacking of responses.
	// eg two login attempts using the same requestID can conflict within
	// the time period they are not yet completed.
	status, output, requestID, err := svc.digitwinRouter.HandleInvokeAction(
		rp.ClientID, rp.ThingID, rp.Name, rp.Data, rp.RequestID, rp.CLCID)

	// there are 3 possible results:
	// on status completed; return output
	// on status failed: return http ok with RequestProgress containing error
	// on status other: return  RequestProgress object with progress
	//
	// This means that the result is either out or a RequestProgress object
	// Forms will have added:
	// ```
	//  "additionalResponses": [{
	//                    "success": false,
	//                    "contentType": "application/json",
	//                    "schema": "RequestProgress"
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
	replyHeader.Set(hubclient.RequestIDHeader, requestID)

	// in case of error include the return data schema
	// TODO: Use schema name from Forms. The action progress schema is in
	// the forms definition as an additional response.
	// right now the only response is an action progress.
	if err != nil {
		replyHeader.Set(hubclient.DataSchemaHeader, "RequestProgress")
		resp := hubclient.RequestProgress{
			RequestID: requestID,
			Progress:  status,
			Error:     err.Error(),
			Reply:     output,
		}
		svc.writeReply(w, resp, nil)
		return
	} else if status != vocab.RequestCompleted {
		// if progress isn't completed then also return the delivery progress
		replyHeader.Set(hubclient.DataSchemaHeader, "RequestProgress")
		// FIXME: shouldn't this be the reply from the action?
		resp := hubclient.RequestProgress{
			RequestID: requestID,
			Progress:  status,
			Reply:     output,
		}
		svc.writeReply(w, resp, nil)
		return
	}
	// TODO: standardize headers
	replyHeader.Set(hubclient.StatusHeader, status)

	// request completed, write output
	svc.writeReply(w, output, nil)
	return
}

// HandleLogin handles a login request and a new session, posted by a consumer
// This uses the configured session authenticator.
func (svc *HttpBinding) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var reply any
	var data interface{}

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &data)
	}
	if err == nil {
		reply, err = svc.digitwinRouter.HandleLogin(data)
	}
	if err != nil {
		slog.Warn("HandleLogin failed:", "err", err.Error())
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}
	// TODO: set client session cookie for browser clients
	//svc.sessionManager.SetSessionCookie(cs.sessionID,token)
	svc.writeReply(w, reply, nil)
}

// HandleLoginRefresh refreshes the auth token using the session authenticator.
// The session authenticator is that of the authn service. This allows testing with a dummy
// authenticator without having to run the authn service.
func (svc *HttpBinding) HandleLoginRefresh(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	clientID, err := httpcontext.GetClientIdFromContext(r)
	if err != nil {
		svc.writeError(w, err, http.StatusBadRequest)
		return
	}
	reply, err := svc.digitwinRouter.HandleLoginRefresh(clientID, rp.Data)
	svc.writeReply(w, reply, err)
}

// HandleLogout ends the session and closes all client connections
func (svc *HttpBinding) HandleLogout(w http.ResponseWriter, r *http.Request) {
	clientID, err := httpcontext.GetClientIdFromContext(r)
	if err != nil {
		// This is an internal error. The middleware session handler would have blocked
		// a request that required a session before getting here.
		err := fmt.Errorf("missing session for request '%s' from '%s'",
			r.RequestURI, r.RemoteAddr)
		slog.Error(err.Error())
		svc.writeError(w, err, http.StatusInternalServerError)
		return
	}
	svc.digitwinRouter.HandleLogout(clientID)
	svc.writeReply(w, nil, nil)
}

// HandlePublishEvent update digitwin with event published by agent
func (svc *HttpBinding) HandlePublishEvent(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// pass the event to the digitwin service for further processing
	requestID := r.Header.Get(hubclient.RequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}
	err = svc.digitwinRouter.HandlePublishEvent(rp.ClientID, rp.ThingID, rp.Name, rp.Data, requestID)
	svc.writeReply(w, nil, err)
}

// HandleQueryAction returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleQueryAction(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	reply, err := svc.digitwinRouter.HandleQueryAction(rp.ClientID, rp.ThingID, rp.Name)
	svc.writeReply(w, reply, err)
}

// HandleQueryAllActions returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleQueryAllActions(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	reply, err := svc.digitwinRouter.HandleQueryAllActions(rp.ClientID, rp.ThingID)
	svc.writeReply(w, reply, err)
}

// HandleReadAllEvents returns a list of latest event values from a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleReadAllEvents(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	reply, err := svc.digitwinRouter.HandleReadAllEvents(rp.ClientID, rp.ThingID)
	svc.writeReply(w, reply, err)
}

// HandleReadAllProperties was added to the top level TD form. Handle it here.
func (svc *HttpBinding) HandleReadAllProperties(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	reply, err := svc.digitwinRouter.HandleReadAllProperties(rp.ClientID, rp.ThingID)
	svc.writeReply(w, reply, err)
}

// HandleReadAllThings returns a list of things in the directory
//
// Query params for paging:
//
//	limit=N, limit the number of results to N TD documents
//	offset=N, skip the first N results in the result
//func (svc *HttpBinding) HandleReadAllThings(w http.ResponseWriter, r *http.Request) {
//	rp, err := subprotocols.GetRequestParams(r)
//	if err != nil {
//		w.WriteHeader(http.StatusUnauthorized)
//		return
//	}
//	slog.Debug("HandleReadAllThings", slog.String("SenderID", rp.ClientID))
//	// this request can simply be turned into an action message for the directory.
//	limit := 100
//	offset := 0
//	if r.URL.Query().Has("limit") {
//		limitStr := r.URL.Query().Get("limit")
//		limit32, _ := strconv.ParseInt(limitStr, 10, 32)
//		limit = int(limit32)
//	}
//	if r.URL.Query().Has("offset") {
//		offsetStr := r.URL.Query().Get("offset")
//		offset32, _ := strconv.ParseInt(offsetStr, 10, 32)
//		offset = int(offset32)
//	}
//	thingsList, err := svc.dtwService.DirSvc.ReadAllTDs(rp.ClientID,
//		digitwin.DirectoryReadAllTDsArgs{Offset: offset, Limit: limit})
//	if err != nil {
//		svc.writeError(w, err, 0)
//		return
//	}
//	svc.writeReply(w, thingsList)
//}

// HandleReadEvent returns the latest event value from a Thing
// Parameters: {thingID}, {name}
func (svc *HttpBinding) HandleReadEvent(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	slog.Debug("HandleReadEvent", slog.String("SenderID", rp.ClientID))
	reply, err := svc.digitwinRouter.HandleReadEvent(rp.ClientID, rp.ThingID, rp.Name)
	svc.writeReply(w, reply, err)
}

func (svc *HttpBinding) HandleReadProperty(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	slog.Debug("HandleReadProperty", slog.String("SenderID", rp.ClientID))
	reply, err := svc.digitwinRouter.HandleReadProperty(rp.ClientID, rp.ThingID, rp.Name)
	svc.writeReply(w, reply, err)
}

// HandleReadTD returns the TD of a thing in the directory
// URL parameter {thingID}
func (svc *HttpBinding) HandleReadTD(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}
	reply, err := svc.digitwinRouter.HandleReadTD(rp.ClientID, rp.ThingID)
	svc.writeReply(w, reply, err)
}

// HandleReadAllTDs returns the list of digital twin TDs in the directory.
// this is a REST api for convenience. Consider using directory action instead.
func (svc *HttpBinding) HandleReadAllTDs(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		svc.writeError(w, err, http.StatusUnauthorized)
		return
	}
	reply, err := svc.digitwinRouter.HandleReadAllTDs(rp.ClientID)
	svc.writeReply(w, reply, err)
}

// HandlePublishMultipleProperties agent sends a map with multiple property
func (svc *HttpBinding) HandlePublishMultipleProperties(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	requestID := r.Header.Get(tlsclient.HTTPRequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}
	propMap := make(map[string]any)
	err = utils.Decode(rp.Data, &propMap)
	if err == nil {
		err = svc.digitwinRouter.HandlePublishMultipleProperties(rp.ClientID, rp.ThingID, propMap, requestID)
	}
	svc.writeReply(w, nil, err)
}

// HandlePublishProperty agent sends single or multiple property updates
func (svc *HttpBinding) HandlePublishProperty(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	requestID := r.Header.Get(tlsclient.HTTPRequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}
	err = svc.digitwinRouter.HandlePublishProperty(rp.ClientID, rp.ThingID, rp.Name, rp.Data, requestID)
	svc.writeReply(w, nil, err)
}

// HandlePublishTD agent sends a new TD document
func (svc *HttpBinding) HandlePublishTD(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	err = svc.digitwinRouter.HandlePublishTD(rp.ClientID, rp.Data)
	svc.writeReply(w, nil, err)
}

// HandleWriteProperty consumer requests to update a Thing property
func (svc *HttpBinding) HandleWriteProperty(w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	status, requestID, err := svc.digitwinRouter.HandleWriteProperty(
		rp.ClientID, rp.ThingID, rp.Name, rp.Data)

	w.Header().Set(tlsclient.HTTPRequestIDHeader, requestID)
	svc.writeReply(w, status, nil)
}
