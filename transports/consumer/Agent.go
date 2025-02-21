package consumer

import (
	"fmt"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/clients"
	"github.com/hiveot/hub/wot"
	"sync/atomic"
	"time"
)

// Agent provides the messaging functions needed by hub agents.
// Agents are also consumers as they are able to invoke services.
//
// Hub agents receive requests and return responses.
// The underlying transport protocol binding handles subscription.
type Agent struct {
	Consumer

	// the application's request handler set with SetRequestHandler
	// intended for sub-protocols that can receive requests. (agents)
	appRequestHandlerPtr atomic.Pointer[transports.RequestHandler]
}

// OnRequest passes a request to the application request handler and returns the response.
// Handler must be set by agent subclasses during init.
// This logs an error if no agent handler is set.
func (ag *Agent) onRequest(
	req *transports.RequestMessage, c transports.IConnection) *transports.ResponseMessage {

	// handle requests if any
	hPtr := ag.appRequestHandlerPtr.Load()
	if hPtr == nil {
		err := fmt.Errorf("Received request but no handler is set")
		resp := req.CreateResponse(nil, err)
		return resp
	}
	resp := (*hPtr)(req, c)
	return resp
}

// PubEvent helper for agents to send an event to subscribers.
// This sends a subscription response message with status running.
//
// The underlying transport protocol binding handles the subscription mechanism
// as the agent itself doesn't track subscriptions.
func (ag *Agent) PubEvent(thingID string, name string, value any) error {
	// This is a response to subscription request.
	// for now assume this is a hub connection and the hub wants all events
	resp := transports.NewResponseMessage(
		wot.OpSubscribeEvent, thingID, name, value, nil, "")

	return ag.cc.SendResponse(resp)
}

// PubProperty helper for agents to publish a property value update to observers.
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperty(thingID string, name string, value any) error {
	// This is a response to an observation request.
	// send the property update as a response to the observe request
	resp := transports.NewResponseMessage(
		wot.OpObserveProperty, thingID, name, value, nil, "")
	return ag.cc.SendResponse(resp)
}

// PubProperties helper for agents to publish a map of property values
//
// The underlying transport protocol binding handles the subscription mechanism.
func (ag *Agent) PubProperties(thingID string, propMap map[string]any) error {

	// Implicit rule: if no name is provided the data is a map
	// the transport adds the correlationID of the subscription.
	resp := transports.NewResponseMessage(
		wot.OpObserveAllProperties, thingID, "", propMap, nil, "")

	return ag.cc.SendResponse(resp)
}

// PubTD helper for agents to publish an update of a TD in the directory
//
// This sends the update-td operation over the connection. The consumer must handle
// it accordingly.
// This is likely to change to better fit the WoT directory specification but
// the API should remain unchanged.
//
// FIXME: Specification:
// https://www.w3.org/TR/wot-discovery/#exploration-td-type-thingdirectory
// > PUT /things/{id}   payload TD JSON; returns 201
// > GET /things/{id}
//func (ag *Agent) PubTD(td *td.TD) error {
//	// TD is sent as JSON
//	tdJson, _ := jsoniter.MarshalToString(td)
//
//	// send a request to the directory to update the TD. The protocol binding will convert
//	// this requires a directory client
//	// option 1: from discovery using HTTP  put /things/{id}
//	// option 2: from protocol binding?
//
//	// this to the appropriate messaging.
//	// * HTTP-Basic bindings uses PUT TD /things/{id}
//	// * HiveOT WSS binding passes it as-is
//	//
//	return ag.Rpc(wot.HTOpUpdateTD, td.ID, "", tdJson, nil)
//}

// SetRequestHandler set the application handler for incoming requests
func (ag *Agent) SetRequestHandler(cb transports.RequestHandler) {
	if cb == nil {
		ag.appRequestHandlerPtr.Store(nil)
	} else {
		ag.appRequestHandlerPtr.Store(&cb)
	}
}

// NewAgent creates a new agent instance for serving requests and sending responses.
// Since agents are also consumers, they can also send requests and receive responses.
//
// Agents can be connected to when running a server or connect to a hub or gateway as client.
//
// This is a wrapper around the ClientConnection that provides WoT response messages
// publishing properties and events to subscribers and publishing a TD.
func NewAgent(cc transports.IConnection,
	reqHandler transports.RequestHandler,
	respHandler transports.ResponseHandler,
	connHandler transports.ConnectionHandler,
	timeout time.Duration) *Agent {

	if timeout == 0 {
		timeout = clients.DefaultTimeout
	}

	//consumer := NewConsumer(cc, respHandler, connHandler, timeout)
	agent := Agent{
		Consumer: Consumer{
			cc:         cc,
			rnrChan:    NewRnRChan(),
			rpcTimeout: timeout,
		},
	}
	agent.SetConnectHandler(connHandler)
	agent.SetRequestHandler(reqHandler)
	agent.SetResponseHandler(respHandler)
	cc.SetResponseHandler(agent.onResponse)
	cc.SetConnectHandler(agent.onConnect)
	cc.SetRequestHandler(agent.onRequest)
	return &agent
}
