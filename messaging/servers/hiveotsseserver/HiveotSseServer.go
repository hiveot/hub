package hiveotsseserver

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
	"github.com/hiveot/hub/wot/td"
)

const (
	DefaultHiveotSsePath = "/hiveot/sse"

	// DefaultHiveotPostRequestHRef HTTP endpoint that accepts HiveOT RequestMessage envelopes
	DefaultHiveotPostRequestHRef = "/hiveot/request"

	// DefaultHiveotPostResponseHRef HTTP endpoint that accepts HiveOT ResponseMessage envelopes
	DefaultHiveotPostResponseHRef = "/hiveot/response"

	// DefaultHiveotPostNotificationHRef HTTP endpoint that accepts HiveOT NotificationMessage envelopes
	DefaultHiveotPostNotificationHRef = "/hiveot/notification"

	SSEOpConnect    = "sse-connect"
	HiveotSSESchema = "sse"
)

// HiveotSseServer is a protocol binding transport server for both HTTP, and the
// SSE-SC sub-protocol. SSE-SC is refers to a single-connection SSE protocol.
// This protocol supports full asynchronous messaging using http/SSE but is not a WoT standard.
//
// Note: The use of SSE is optional. This binding still serves an important role as the
// HTTPp protocol binding, and provides login and token refresh endpoints.
//
// This SSE protocol implementation is not a WoT specified protocol. It is however
// easier and more efficient to use, requiring only a single connection.
// It uses the hiveot RequestMessage and ResponseMessage envelopes for all messaging, or
// an alternative message converter can be provided to support a different message
// envelope format.
//
// This can be used with any golang HTTP server, including the http-basic or
// websocket http server as long as it can register routes.
//
// Usages:
//
//  1. Thing Agents that run servers. For example the HiveOT Hub. The agent
//     serves HTTP/SSE cm from consumers. Requests are received over HTTP
//     and asynchronous responses are sent back over SSE. HTTP requests and SSE
//     cm must carry the same 'cid' to correlate HTTP requests with the
//     SSE return channel from the same client.
//     The HiveOT Hub uses this as part of multiple servers that serve the
//     digital twin repository content.
//
//  2. Consumers that run servers. For example the Hub is a consumer of Thing agents
//     that connect to the Hub. Since the connection is reversed, the requests are
//     now sent over SSE to the Hub while the response is sent as a HTTP post to the hub.
//
//  3. An agent/consumer hybrid that runs a server. For example, the HiveOT Hub.
//     Another Thing agent or service connect to the Hub to receive requests and
//     at the same time can send consumer requests over http and receive responses
//     over SSE.
//
// # Note that the direction of connection is independent of this transport does not determine the
//
// All SSE messages use the 'event' and 'data' field as per SSE standard. The event
// field contains the operation while the data field contains the RequestMessage
// or ResponseMessage envelope.
//
// SSE 'event' field contains the request or response message type, indicating the
// message payload.
type HiveotSseServer struct {

	// manage the incoming SSE cm
	cm *connections.ConnectionManager

	// mutex for updating cm
	mux sync.RWMutex

	// registered handler of incoming cm
	serverConnectHandler messaging.ConnectionHandler

	// The connection address for subscription and URL to connect using SSE
	connectAddr string
	connectURL  string

	// The router to add to on start
	router chi.Router

	// registered handler of incoming requests (which return a reply)
	serverNotificationHandler messaging.NotificationHandler
	// registered handler of incoming requests (which return a reply)
	serverRequestHandler messaging.RequestHandler
	// registered handler of incoming responses (which sends a reply to the request sender)
	serverResponseHandler messaging.ResponseHandler

	// The SSE connection path
	ssePath string
}

// AddRoutes adds routes to the HTTP server for connecting to SSE, (Un)Subscribe,
// and (Un)Observe using hiveot RequestMessage and ResponseMessage envelopes.
// This

// AddTDForms for connecting to SSE, Subscribe, Observe, Send Requests, read and query
// using hiveot RequestMessage and ResponseMessage envelopes.
func (srv *HiveotSseServer) AddTDForms(tdi *td.TD, includeAffordances bool) {

	// TODO: add the hiveot http endpoints
	//srv.httpBasicServer.AddOps()
	// forms are handled through the http binding
	//return srv.httpBasicServer.AddTDForms(tdi, includeAffordances)
}

func (srv *HiveotSseServer) CloseAll() {
	srv.cm.CloseAll()
}

// CloseAllClientConnections close all cm from the given client.
// Intended to close cm after a logout.
func (srv *HiveotSseServer) CloseAllClientConnections(clientID string) {
	srv.cm.ForEachConnection(func(c messaging.IServerConnection) {
		cinfo := c.GetConnectionInfo()
		if cinfo.ClientID == clientID {
			c.Disconnect()
		}
	})
}

// GetConnectURL returns SSE connection URL of the server
// This uses the custom 'ssesc' schema which is non-wot compatible.
func (srv *HiveotSseServer) GetConnectURL() string {
	return srv.connectURL
}

func (srv *HiveotSseServer) GetProtocolType() string {
	return messaging.ProtocolTypeHiveotSSE
}

// GetConnectionByConnectionID returns the connection with the given connection ID
func (srv *HiveotSseServer) GetConnectionByConnectionID(clientID, cid string) messaging.IConnection {
	return srv.cm.GetConnectionByConnectionID(clientID, cid)
}

// GetConnectionByClientID returns the connection with the given client ID
func (srv *HiveotSseServer) GetConnectionByClientID(agentID string) messaging.IConnection {
	return srv.cm.GetConnectionByClientID(agentID)
}

// GetSseConnection returns the SSE Connection with the given ID
// This returns nil if not found or if the connectionID is not
func (srv *HiveotSseServer) GetSseConnection(clientID, connectionID string) *HiveotSseServerConnection {
	c := srv.cm.GetConnectionByConnectionID(clientID, connectionID)
	if c == nil {
		return nil
	}
	sseConn, isValid := c.(*HiveotSseServerConnection)
	if !isValid {
		return nil
	}
	return sseConn
}

// SendNotification sends a property update or event notification message to subscribers
func (srv *HiveotSseServer) SendNotification(msg *messaging.NotificationMessage) {
	// pass the notification to all subscribed clients
	srv.cm.ForEachConnection(func(c messaging.IServerConnection) {
		_ = c.SendNotification(msg)
	})
	return
}

// Start listening for incoming SSE connections
func (srv *HiveotSseServer) Start() error {
	slog.Info("Starting SSE server, Listening on: " + srv.GetConnectURL())

	// TODO: detect if already listening
	// Add the routes used in SSE connection and subscription requests
	srv.CreateRoutes(srv.ssePath, srv.router)

	return nil
}

// Stop disconnects clients and remove connection listening
func (srv *HiveotSseServer) Stop() {
	//Close all incoming SSE cm
	srv.cm.CloseAll()
	srv.DeleteRoutes(srv.ssePath, srv.router)
}

// NewHiveotSseServer returns a new SSE-SC sub-protocol binding.
// This is only a 1-way binding that adds an SSE based return channel to the http binding.
//
// This adds http methods for (un)subscribing to events and properties and
// adds new cm to the connection manager for callbacks.
//
// router is the protected route that serves sse connections on the ssePath
//
// This fails if no ssePath is provided
func NewHiveotSseServer(
	connectAddr string, ssePath string,
	router chi.Router,
	handleConnect messaging.ConnectionHandler,
	handleNotification messaging.NotificationHandler,
	handleRequest messaging.RequestHandler,
	handleResponse messaging.ResponseHandler,
) *HiveotSseServer {
	if ssePath == "" {
		return nil
	}
	connectURL := fmt.Sprintf("%s://%s%s", HiveotSSESchema, connectAddr, ssePath)
	srv := &HiveotSseServer{
		ssePath:                   ssePath,
		router:                    router,
		cm:                        connections.NewConnectionManager(),
		serverConnectHandler:      handleConnect,
		serverNotificationHandler: handleNotification,
		serverRequestHandler:      handleRequest,
		serverResponseHandler:     handleResponse,
		connectURL:                connectURL,
	}
	return srv
}
