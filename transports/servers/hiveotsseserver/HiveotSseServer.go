package hiveotsseserver

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot/td"
	"net/url"
	"sync"
)

const (
	DefaultHiveotSsePath = "/hiveot/sse"

	// DefaultHiveotPostRequestHRef HTTP endpoint that accepts HiveOT RequestMessage envelopes
	DefaultHiveotPostRequestHRef = "/hiveot/request"

	// DefaultHiveotPostResponseHRef HTTP endpoint that accepts HiveOT ResponseMessage envelopes
	DefaultHiveotPostResponseHRef = "/hiveot/response"

	SSEOpConnect    = "sse-connect"
	HiveotSSESchema = "sse"
)

// HiveotSseServer is a protocol binding transport server of http for the SSE-SC
// Single-Connection protocol. This protocol supports full asynchronous messaging
// over http and SSE.
//
// This is not a WoT specified protocol but is arguably easier to use. It uses
// the hiveot RequestMessage and ResponseMessage envelopes for all messaging, or
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

	httpTransport *httpserver.HttpTransportServer

	// mutex for updating cm
	mux sync.RWMutex

	// registered handler of incoming cm
	serverConnectHandler transports.ConnectionHandler

	// The listening path
	ssePath string

	// registered handler of incoming requests (which return a reply)
	serverRequestHandler transports.RequestHandler
	// registered handler of incoming responses (which sends a reply to the request sender)
	serverResponseHandler transports.ResponseHandler
}

// AddRoutes adds routes to the HTTP server for connecting to SSE, (Un)Subscribe,
// and (Un)Observe using hiveot RequestMessage and ResponseMessage envelopes.
// This

// AddTDForms for connecting to SSE, Subscribe, Observe, Send Requests, read and query
// using hiveot RequestMessage and ResponseMessage envelopes.
func (srv *HiveotSseServer) AddTDForms(tdi *td.TD) error {

	// TODO: add the hiveot http endpoints
	//srv.httpTransport.AddOps()
	// forms are handled through the http binding
	//return srv.httpTransport.AddTDForms(tdi)
	return nil
}

func (srv *HiveotSseServer) CloseAll() {
	srv.cm.CloseAll()
}

// CloseAllClientConnections close all cm from the given client.
// Intended to close cm after a logout.
func (srv *HiveotSseServer) CloseAllClientConnections(clientID string) {
	srv.cm.ForEachConnection(func(c transports.IServerConnection) {
		cinfo := c.GetConnectionInfo()
		if cinfo.ClientID == clientID {
			c.Disconnect()
		}
	})
}

// GetConnectURL returns SSE connection URL of the server
// This uses the custom 'ssesc' schema which is non-wot compatible.
func (srv *HiveotSseServer) GetConnectURL() string {
	httpURL := srv.httpTransport.GetConnectURL()
	parts, err := url.Parse(httpURL)
	if err != nil {
		return ""
	}
	ssePath := fmt.Sprintf("%s://%s%s", HiveotSSESchema, parts.Host, srv.ssePath)
	return ssePath
}
func (srv *HiveotSseServer) GetProtocolType() string {
	return transports.ProtocolTypeHiveotSSE
}

// GetConnectionByConnectionID returns the connection with the given connection ID
func (srv *HiveotSseServer) GetConnectionByConnectionID(clientID, cid string) transports.IConnection {
	return srv.cm.GetConnectionByConnectionID(clientID, cid)
}

// GetConnectionByClientID returns the connection with the given client ID
func (srv *HiveotSseServer) GetConnectionByClientID(agentID string) transports.IConnection {
	return srv.cm.GetConnectionByClientID(agentID)
}

// GetForm returns a new SSE form for the given operation
// this returns the http form
func (srv *HiveotSseServer) GetForm(op, thingID, name string) *td.Form {
	// forms are handled through the http binding
	return srv.httpTransport.GetForm(op, thingID, name)
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

// SendNotification sends a property update or event response message to subscribers
func (srv *HiveotSseServer) SendNotification(msg *transports.ResponseMessage) {
	// pass the response to all subscribed cm
	// FIXME: track cm
	srv.cm.ForEachConnection(func(c transports.IServerConnection) {
		c.SendNotification(*msg)
	})
}

func (srv *HiveotSseServer) Stop() {
	//Close all incoming SSE cm
	srv.cm.CloseAll()
}

// StartHiveotSseServer returns a new SSE-SC sub-protocol binding.
// This is only a 1-way binding that adds an SSE based return channel to the http binding.
//
// This adds http methods for (un)subscribing to events and properties and
// adds new cm to the connection manager for callbacks.
//
// This fails if no ssePath is provided
func StartHiveotSseServer(
	ssePath string,
	httpTransport *httpserver.HttpTransportServer,
	handleConnect transports.ConnectionHandler,
	handleRequest transports.RequestHandler,
	handleResponse transports.ResponseHandler,
) *HiveotSseServer {
	if ssePath == "" {
		return nil
	}
	srv := &HiveotSseServer{
		cm:                    connections.NewConnectionManager(),
		serverConnectHandler:  handleConnect,
		serverRequestHandler:  handleRequest,
		serverResponseHandler: handleResponse,
		ssePath:               ssePath,
		httpTransport:         httpTransport,
	}
	// Add the routes used in SSE connection and subscription requests
	srv.CreateRoutes()
	return srv
}
