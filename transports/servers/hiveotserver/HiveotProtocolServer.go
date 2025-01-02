package hiveotserver

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
	"sync"
)

// HiveotProtocolServer is an application protocol server that runs on top of existing
// transport protocols.
// This is (IMHO) the protocol that would simplify the use of WoT and forms as it
// only has three messages: request, response and notifications.
//
// This protocol server currently hooks into the http/ssesc, wss and mqtt transport
// servers and uses these as transports. Note that this involves quite a bit of
// mapping between message formats.
//
// In future this is intended to become its own server using simple transports
// over http, wss, mqtt containing just the 3 message types.
//
// Currently this protocol is only used by agents as there are no operations and
// forms supporting agents in WoT.
//
// Subscription is requested through the subscribe/observe operations in the request
// message. This is applied to the return channels of the underlying transport protocols.
type HiveotProtocolServer struct {
	authenticator transports.IAuthenticator

	// connection manager to add/remove connections
	cm *connections.ConnectionManager

	// The http server to register the endpoints with
	httpTransport *httpserver.HttpTransportServer

	// registered handler of received notifications (sent by agents)
	serverNotificationHandler transports.ServerNotificationHandler
	// registered handler of requests (which return a reply)
	serverRequestHandler transports.ServerRequestHandler
	// registered handler of responses (which sends a reply to the request sender)
	serverResponseHandler transports.ServerResponseHandler

	// mutex for updating connections
	mux sync.RWMutex
}

func (svc *HiveotProtocolServer) AddTDForms(tdi *td.TD) error {
	return nil
}

// GetForm returns a form for the given operation
func (svc *HiveotProtocolServer) GetForm(op string) *td.Form {
	// forms are handled through the http binding
	return nil
}

// GetConnectURL returns SSE connection path of the server
func (svc *HiveotProtocolServer) GetConnectURL() string {
	return svc.httpTransport.GetConnectURL()
}

// HandleHttpRequest use the hiveot message envelope for the underlying http transport.
// The payload is a RequestMessage envelope that contains all request information.
//
// The connectionID is used to send the reply to and to subscribe/observe.
func (svc *HiveotProtocolServer) HandleHttpRequest(w http.ResponseWriter, r *http.Request) {

	var response transports.ResponseMessage
	clientID, connID, reqJson, err := httpserver.GetHiveotParams(r)
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	request := transports.RequestMessage{}
	err = jsoniter.Unmarshal(reqJson, &request)
	// enforce a correct sender ID
	request.SenderID = clientID

	// ping and subscription are handled internally
	switch request.Operation {
	case wot.HTOpPing:
		// regular http server returns with pong -
		// used only when no sub-protocol is used as return channel
		response = request.CreateResponse("pong", nil)
	case wot.HTOpRefresh:
		oldToken := request.ToString()
		newToken, err := svc.authenticator.RefreshToken(
			request.SenderID, request.SenderID, oldToken)
		response = request.CreateResponse(newToken, err)
	case wot.OpObserveProperty, wot.OpObserveAllProperties:
		c := svc.cm.GetConnectionByConnectionID(connID)
		if c != nil {
			c.ObserveProperty(request.ThingID, request.Name)
		} else {
			err = fmt.Errorf("Observe failed. Connection '%s' not found", connID)
		}
		response = request.CreateResponse(nil, err)
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		c := svc.cm.GetConnectionByConnectionID(connID)
		if c != nil {
			c.SubscribeEvent(request.ThingID, request.Name)
		} else {
			err = fmt.Errorf("Subscribe failed. Connection '%s' not found", connID)
		}
		response = request.CreateResponse(nil, err)
	case wot.OpUnobserveProperty, wot.OpUnobserveAllProperties:
		c := svc.cm.GetConnectionByConnectionID(connID)
		if c != nil {
			c.UnobserveProperty(request.ThingID, request.Name)
		} else {
			err = fmt.Errorf("Unobserve failed. Connection '%s' not found", connID)
		}
		response = request.CreateResponse(nil, err)
	case wot.OpUnsubscribeEvent, wot.OpUnsubscribeAllEvents:
		c := svc.cm.GetConnectionByConnectionID(connID)
		if c != nil {
			c.UnsubscribeEvent(request.ThingID, request.Name)
		} else {
			err = fmt.Errorf("Unsubscribe failed. Connection '%s' not found", connID)
		}
		response = request.CreateResponse(nil, err)

	default: // forward to the request handler
		if svc.serverRequestHandler == nil {
			slog.Error("No request handler registered")
			panic("no request handler registered")
		} else {
			// forward the request to the internal handler for further processing.
			// If a result is available immediately, it will be embedded into the http
			// response body, otherwise a status pending is returned.
			// a return channel with the same connection ID is required.
			response = svc.serverRequestHandler(request, connID)
		}
	}
	// The hiveot server handler returns the ResponseMessage format. The sender
	// should be aware of this.
	respJson, _ := jsoniter.Marshal(response)
	_, _ = w.Write(respJson)
}

// HandleHttpResponse uses the hiveot message envelope handler for handling
// agent responses and forward them to the registered handler.
// The payload is a ResponseMessage envelope that contains all response information.
// Intended for agents to send responses asynchronously.
func (svc *HiveotProtocolServer) HandleHttpResponse(w http.ResponseWriter, r *http.Request) {
	var response transports.ResponseMessage

	clientID, connID, payload, err := httpserver.GetHiveotParams(r)
	_ = clientID
	_ = connID
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = jsoniter.Unmarshal(payload, &response)
	// server MUST set sender to the authenticated client
	response.SenderID = clientID

	if svc.serverResponseHandler == nil {
		err = fmt.Errorf("No server response handler registered. Response ignored.")
	} else {
		// forward the response to the internal handler for further processing.
		err = svc.serverResponseHandler(response)
	}
	// response handling is complete, there is no output
	w.WriteHeader(http.StatusOK)
}

// HandleHttpNotification uses the hiveot message envelope handler for handling
// agent responses and forward them to the registered handler.
//
// The payload is a NotificationMessage envelope that contains all information.
// Intended for agents to publish notifications.
func (svc *HiveotProtocolServer) HandleHttpNotification(w http.ResponseWriter, r *http.Request) {
	var notification transports.NotificationMessage

	clientID, connID, notifJson, err := httpserver.GetHiveotParams(r)
	_ = clientID
	_ = connID
	if err != nil {
		slog.Error(err.Error())
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	err = jsoniter.Unmarshal(notifJson, &notification)
	notification.SenderID = clientID

	if svc.serverNotificationHandler == nil {
		err = fmt.Errorf("No server notification handler registered. Notification ignored.")
		slog.Error(err.Error())
	} else {
		// forward the response to the internal handler for further processing.
		svc.serverNotificationHandler(notification)
	}
	// notification handling is complete, there is no output
	w.WriteHeader(http.StatusOK)
}

// SendNotification broadcast an event or property change to subscribers clients
func (svc *HiveotProtocolServer) SendNotification(notification transports.NotificationMessage) {
	panic("how did we get here?")
}

func (svc *HiveotProtocolServer) Stop() {
	//Close all incoming SSE connections
	svc.cm.CloseAll()
}

// StartHiveotProtocolServer returns a new application protocol binding that
// utilizes the given http transport protocol.
//
// TODO: Using the given transport server is a temporary messy hook so it can
// register the hiveot http endpoints for Requests, Responses and Notifications. Once the
// dust settles this needs cleanup.
//
// TODO: also handle WSS and MQTT transported messages
//
// This adds http methods for (un)subscribing to events and properties and
// add new connections to the connection manager for callbacks.
func StartHiveotProtocolServer(
	authenticator transports.IAuthenticator,
	cm *connections.ConnectionManager,
	httpTransport *httpserver.HttpTransportServer,
	handleNotification transports.ServerNotificationHandler,
	handleRequest transports.ServerRequestHandler,
	handleResponse transports.ServerResponseHandler,
) *HiveotProtocolServer {
	b := &HiveotProtocolServer{
		authenticator:             authenticator,
		cm:                        cm,
		httpTransport:             httpTransport,
		serverNotificationHandler: handleNotification,
		serverRequestHandler:      handleRequest,
		serverResponseHandler:     handleResponse,
	}
	httpTransport.AddOps(nil, []string{"request"},
		http.MethodPost, httpserver.HiveOTPostRequestHRef, b.HandleHttpRequest)
	httpTransport.AddOps(nil, []string{"response"},
		http.MethodPost, httpserver.HiveOTPostResponseHRef, b.HandleHttpResponse)
	httpTransport.AddOps(nil, []string{"notification"},
		http.MethodPost, httpserver.HiveOTPostNotificationHRef, b.HandleHttpNotification)
	return b
}
