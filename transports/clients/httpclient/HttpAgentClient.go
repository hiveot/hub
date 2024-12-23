package httpclient

import (
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

// HttpAgentClient provides additional functions used for agents. Intended
// for use by sub-protocol agents.
// This implements the IAgentTransport interface.
type HttpAgentClient struct {
	consumerTransport *HttpConsumerClient
}

// SendNotification publishes a notification to subscribers
// This returns an error if the notification could not be delivered to the server
func (cl *HttpAgentClient) SendNotification(notification transports.NotificationMessage) error {
	var payload []byte

	// if a form is defined then use it, otherwise fall back to the built-in format
	form := cl.consumerTransport.getForm(notification.Operation)
	href, _ := form.GetHRef()
	method, _ := form.GetMethodName()
	if href == "" || method == "" {
		// use the built-in format
		href = httpserver.GenericHttpHRef
		method = http.MethodPost
	} else if notification.Data != nil {
		payload, _ = jsoniter.Marshal(notification.Data)
	}
	_, _, err := cl.consumerTransport._send(method, href, "", notification.ThingID,
		notification.Name, payload, "")
	return err
}

// SendResponse Agent sends a response to a request.
func (cl *HttpAgentClient) SendResponse(resp transports.ResponseMessage) error {
	var payload []byte

	// if a form is defined then use it, otherwise fall back to the built-in format
	form := cl.consumerTransport.getForm(resp.Operation)
	href, _ := form.GetHRef()
	method, _ := form.GetMethodName()
	if href == "" || method == "" {
		// use the built-in format
		href = httpserver.GenericHttpHRef
		method = http.MethodPost
	}
	// invokeaction response is an action status message. everyone else just returns a payload
	if resp.Operation == wot.OpInvokeAction {
		actionStatus := httpserver.HttpActionStatus{
			Status: resp.Status, // 1:1 conversion
			Output: resp.Output,
			Error:  resp.Error,
			// Note: the Agent HRef is not accessible to the consumer as agents are
			// connected as a client. Instead, the server caches the last known
			// status and provides it in queryaction.
			Href:          "",
			TimeRequested: resp.Received,
			TimeEnded:     resp.Updated,
		}
		payload, _ = jsoniter.Marshal(actionStatus)
	} else if resp.Output != nil {
		payload, _ = jsoniter.Marshal(resp.Output)
	}
	_, _, err := cl.consumerTransport._send(method, href, "", resp.ThingID, resp.Name, payload, resp.RequestID)
	return err
}

func (cl *HttpAgentClient) Init(ct *HttpConsumerClient) {
	cl.consumerTransport = ct
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
