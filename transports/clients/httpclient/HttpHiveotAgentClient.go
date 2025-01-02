// package httpclient with hiveot messaging protocol for use by agents
package httpclient

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

// HttpAgentClient provides functions used for agents. Intended for use together
// with the http consumer client.
// This implements the IAgentTransport interface.
type HttpAgentClient struct {
	consumerTransport *HttpConsumerClient
}

// PubEvent helper for agents to publish an event
// This is short for SendNotification( ... wot.OpEvent ...)
func (cl *HttpAgentClient) PubEvent(thingID string, name string, value any) error {
	notif := transports.NewNotificationMessage(wot.HTOpEvent, thingID, name, value)
	return cl.SendNotification(notif)
}

// PubProperty helper for agents to publish a property value update
// This is short for SendNotification( ... wot.OpProperty ...)
func (cl *HttpAgentClient) PubProperty(thingID string, name string, value any) error {

	notif := transports.NewNotificationMessage(wot.HTOpUpdateProperty, thingID, name, value)
	return cl.SendNotification(notif)
}

// PubProperties helper for agents to publish a map of property values
// TODO: support multiple properties?
func (cl *HttpAgentClient) PubProperties(thingID string, propMap map[string]any) error {

	notif := transports.NewNotificationMessage(wot.HTOpUpdateMultipleProperties, thingID, "", propMap)
	err := cl.SendNotification(notif)
	return err
}

// PubTD helper for agents to publish a TD update
// This is short for SendNotification( ... wot.HTOpTD ...)
func (cl *HttpAgentClient) PubTD(td *td.TD) error {
	// JSON must be encoded as string
	tdJson, _ := jsoniter.MarshalToString(td)
	notif := transports.NewNotificationMessage(wot.HTOpUpdateTD, td.ID, "", tdJson)
	return cl.SendNotification(notif)
}

// SendNotification publishes a notification to subscribers using the hiveot protocol
// This returns an error if the notification could not be delivered to the server
func (cl *HttpAgentClient) SendNotification(notification transports.NotificationMessage) error {
	var payload []byte

	// WoT HTTP Basic protocol doesn't support agents so use the hiveot protocol
	href := httpserver.HiveOTPostNotificationHRef
	method := http.MethodPost

	payload, _ = jsoniter.Marshal(notification)
	_, _, err := cl.consumerTransport._send(method, href, payload, "")
	return err
}

// HttpHiveotPubRequest publishes a request message to the server. - not used
//
// This doesn't wait for a response and immediately returns.
// Intended to substitute the consumer http post request message with the hiveot
// protocol version, which doesn't use forms.
//
// TODO: move this method into a dedicated hiveot protocol binding which uses
// http,sse,wss,mqtt as mere transports. It won't be interoperable with anything
// so do this only for performance and code comparison. It should be simpler and faster.
//
// this currently isn't used as the http pubrequest is used
// the server side needs to support requests for things like subscription before
// this can be used.
func (cl *HttpAgentClient) HttpHiveotPubRequest(req transports.RequestMessage) error {
	var payload []byte

	// WoT HTTP Basic protocol doesn't support agents so use the hiveot protocol
	href := httpserver.HiveOTPostRequestHRef
	method := http.MethodPost

	payload, _ = jsoniter.Marshal(req)

	outputRaw, headers, err := cl.consumerTransport._send(method, href, payload, "")
	_ = headers
	// optional synchronous response
	if err == nil && outputRaw != nil {
		resp := transports.ResponseMessage{}
		err = jsoniter.Unmarshal(outputRaw, &resp)
		go cl.consumerTransport.OnResponse(resp)
	}

	return err
}

//
//// SendRequest Agent sends a request using the hiveot protocol
//func (cl *HttpAgentClient) SendRequest(req transports.RequestMessage, waitForCompletion bool) (
//	transports.ResponseMessage, error) {
//
//	var resp transports.ResponseMessage
//
//	// Responses do not have forms in WoT - use the generic server response path
//	href := httpserver.HiveOTPostRequestHRef
//	method := http.MethodPost
//
//	payload, _ := jsoniter.Marshal(req)
//	respData, _, err := cl.consumerTransport._send(
//		method, href, "", "", "", payload, resp.CorrelationID)
//
//	if err == nil {
//		err = jsoniter.Unmarshal(respData, &resp)
//	} else {
//		// make sure the response has the critical fields
//		resp.Operation = req.Operation
//		resp.CorrelationID = req.CorrelationID
//		resp.Status = transports.StatusFailed
//		resp.Error = err.Error()
//	}
//	if err == nil && resp.Status == transports.StatusFailed && resp.Error != "" {
//		err = errors.New(resp.Error)
//	}
//
//	// FIXME: wait for completion or split the send
//	cl.BasePubRequest
//
//	return resp, err
//}

// SendResponse Agent sends a response using the hiveot protocol
func (cl *HttpAgentClient) SendResponse(resp transports.ResponseMessage) error {
	var payload []byte

	// Responses do not have forms in WoT - use the generic server response path
	href := httpserver.HiveOTPostResponseHRef
	method := http.MethodPost

	payload, _ = jsoniter.Marshal(resp)
	_, _, err := cl.consumerTransport._send(method, href, payload, resp.CorrelationID)
	return err
}

func (cl *HttpAgentClient) Init(ct *HttpConsumerClient) {
	cl.consumerTransport = ct

	//cl.consumerTransport.BasePubRequest = cl.HttpHiveotPubRequest

}

// NewHttpAgentTransport creates a new instance of the http agent client
//
// This is based on the HttpConsumerClient.
//
//	fullURL of server to connect to, including the schema
//	clientID to connect as; for logging and ConnectWithPassword. It is ignored if auth token is used.
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	timeout for waiting for response. 0 to use the default.
func NewHttpAgentTransport(ct *HttpConsumerClient) *HttpAgentClient {

	cl := HttpAgentClient{}
	cl.Init(ct)
	return &cl
}
