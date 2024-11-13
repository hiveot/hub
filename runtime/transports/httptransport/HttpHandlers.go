// Package httptransport with handlers for the http protocol
package httptransport

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/transports/httptransport/httpcontext"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"io"
	"log/slog"
	"net/http"
)

// Http binding with form handler methods

func (svc *HttpBinding) _handleMessage(op string, w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	// pass the event to the digitwin service for further processing
	requestID := r.Header.Get(hubclient.RequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}
	msg := hubclient.NewThingMessage(op, rp.ThingID, rp.Name, rp.Data, rp.ClientID)
	svc.digitwinRouter.HandleMessage(msg)
	svc.writeReply(w, nil, nil)
}

// _handleRequest provides the boilerplate code for reading headers,
// unmarshalling the arguments and returning a response. If no immediate
// result is available this returns an alternative RequestStatus result object.
func (svc *HttpBinding) _handleRequest(op string, w http.ResponseWriter, r *http.Request) {
	rp, err := httpcontext.GetRequestParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// an action request should have a cid when used with SSE.
	// for now just warn if its missing
	if r.Header.Get(hubclient.ConnectionIDHeader) == "" {
		slog.Warn("InvokeAction request without 'cid' header. This is needed for the SSE subprotocol",
			"clientID", rp.ClientID)
	}

	// pass the event to the digitwin service for further processing
	requestID := r.Header.Get(hubclient.RequestIDHeader)
	if requestID == "" {
		requestID = shortid.MustGenerate()
	}

	// there are 3 possible results:
	// on status completed; return output
	// on status failed: return http ok with RequestStatus containing error
	// on status other: return  RequestStatus object with progress
	//
	// This means that the result is either out or a RequestStatus object
	// Forms will have added:
	// ```
	//  "additionalResponses": [{
	//                    "success": false,
	//                    "contentType": "application/json",
	//                    "schema": "RequestStatus"
	//                }]
	//```
	msg := hubclient.NewThingMessage(op, rp.ThingID, rp.Name, rp.Data, rp.ClientID)
	msg.RequestID = requestID
	stat := svc.digitwinRouter.HandleRequest(msg, rp.CLCID)

	//
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
		replyHeader.Set(hubclient.StatusHeader, vocab.RequestFailed)
		svc.writeReply(w, nil, err)
		return
	} else if stat.Error != "" {
		replyHeader.Set(hubclient.StatusHeader, vocab.RequestFailed)
		svc.writeReply(w, nil, errors.New(stat.Error))
		return
	} else if stat.Progress != vocab.RequestCompleted {
		// if progress isn't completed then also return the delivery progress
		replyHeader.Set(hubclient.StatusHeader, stat.Progress)
		replyHeader.Set(hubclient.DataSchemaHeader, "RequestStatus")
		svc.writeReply(w, stat, nil)
		return
	}
	// progress is complete, return the default output
	svc.writeReply(w, stat.Output, nil)
}

// HandlePublishRequestProgress sends an action progress update message to the digital twin
func (svc *HttpBinding) HandlePublishRequestProgress(w http.ResponseWriter, r *http.Request) {
	svc._handleMessage(vocab.WotOpPublishActionStatus, w, r)
}

// HandleInvokeAction requests an action from the digital twin.
// NOTE: This returns a header with a dataschema if a schema from
// additionalResponses is returned.
//
// The sender must include the connection-id header of the connection it wants to
// receive the response.
func (svc *HttpBinding) HandleInvokeAction(w http.ResponseWriter, r *http.Request) {

	svc._handleRequest(vocab.WotOpInvokeAction, w, r)
}

// HandleLogin handles a login request, posted by a consumer.
//
// This uses the configured session authenticator.
func (svc *HttpBinding) HandleLogin(w http.ResponseWriter, r *http.Request) {
	var reply any
	var data interface{}

	payload, err := io.ReadAll(r.Body)
	if err == nil {
		err = jsoniter.Unmarshal(payload, &data)
	}
	if err == nil {
		//token := svc.authenticator.Login(clientID,password)
		msg := hubclient.NewThingMessage(vocab.HTOpLogin, "", "", data, "")
		stat := svc.digitwinRouter.HandleRequest(msg, "")
		if stat.Error != "" {
			err = errors.New(stat.Error)
		}
		reply = stat.Output
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
	svc._handleRequest(vocab.HTOpRefresh, w, r)
}

// HandleLogout ends the session and closes all client connections
func (svc *HttpBinding) HandleLogout(w http.ResponseWriter, r *http.Request) {
	svc._handleMessage(vocab.HTOpLogout, w, r)
}

// HandlePublishEvent update digitwin with event published by agent
func (svc *HttpBinding) HandlePublishEvent(w http.ResponseWriter, r *http.Request) {
	svc._handleMessage(vocab.WotOpPublishEvent, w, r)
}

// HandleQueryAction returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleQueryAction(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.WotOpQueryAction, w, r)
}

// HandleQueryAllActions returns a list of latest action requests of a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleQueryAllActions(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.WotOpQueryAllActions, w, r)
}

// HandleReadAllEvents returns a list of latest event values from a Thing
// Parameters: thingID
func (svc *HttpBinding) HandleReadAllEvents(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.HTOpReadAllEvents, w, r)
}

// HandleReadAllProperties was added to the top level TD form. Handle it here.
func (svc *HttpBinding) HandleReadAllProperties(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.WotOpReadAllProperties, w, r)
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
	svc._handleRequest(vocab.HTOpReadEvent, w, r)
}

func (svc *HttpBinding) HandleReadProperty(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.WotOpReadProperty, w, r)
}

// HandleReadTD returns the TD of a thing in the directory
// URL parameter {thingID}
func (svc *HttpBinding) HandleReadTD(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.HTOpReadTD, w, r)
}

// HandleReadAllTDs returns the list of digital twin TDs in the directory.
// this is a REST api for convenience. Consider using directory action instead.
func (svc *HttpBinding) HandleReadAllTDs(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.HTOpReadAllTDs, w, r)
}

// HandlePublishMultipleProperties agent sends a map with multiple property
func (svc *HttpBinding) HandlePublishMultipleProperties(w http.ResponseWriter, r *http.Request) {
	svc._handleMessage(vocab.WotOpPublishProperties, w, r)
}

// HandlePublishProperty agent sends single or multiple property updates
func (svc *HttpBinding) HandlePublishProperty(w http.ResponseWriter, r *http.Request) {
	svc._handleMessage(vocab.WotOpPublishProperty, w, r)
}

// HandlePublishTD agent sends a new TD document
func (svc *HttpBinding) HandlePublishTD(w http.ResponseWriter, r *http.Request) {
	svc._handleMessage(vocab.HTOpUpdateTD, w, r)
}

// HandleWriteProperty consumer requests to update a Thing property
func (svc *HttpBinding) HandleWriteProperty(w http.ResponseWriter, r *http.Request) {
	svc._handleRequest(vocab.WotOpWriteProperty, w, r)
}
