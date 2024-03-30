package httpsbinding

// HttpsBindingConfig contains the configuration of the HTTPS server binding,
// including the websocket and the sse configuration.
type HttpsBindingConfig struct {
	// enable websocket support. Default is false.
	EnableWS bool `yaml:"enableWS,omitempty"`

	// enable SSE support. Default is false.
	EnableSSE bool `yaml:"enableSSE,omitempty"`

	// Host is the server address, default is outbound IP address
	Host string `yaml:"host,omitempty"`

	// Port is the server TLS port, default is 444
	// This port handles http, websocket and sse requests
	Port int `yaml:"port,omitempty"`

	// GetActionsPath is the router path for agents to read queued actions
	// Default is "/action/{agentID}/{thingID}"
	GetActionsPath string `yaml:"getActionsPath,omitempty"`

	// GetTDDPath is the router path that handles reading TD documents from the directory
	// Default is "/thing/directory/{agentID}/{thingID}"
	//  {agentID} is the accountID of the agent under which the TDD was published. This is required.
	//  {thingID} is the digital twin ThingID and can be omitted to get the TDDs of all things of the agent
	// This returns a json encoded list of TD documents.
	GetTDDPath string `yaml:"getTDDPath,omitempty"`

	// GetValuesPath is the router path for consumers to read thing values
	// Default is "/thing/values/{agentID}/{thingID}/{key}"
	//  {agentID} is the accountID of the agent under which the TDD was published. This is required.
	//  {thingID} is the digital twin ThingID whose values to read
	//  {key} is the optional key to read. Omit to read all values of the thing.
	// This returns a json encoded map of key-value pairs
	GetValuesPath string `yaml:"getTDDPath,omitempty"`

	// PostActionPath is the router path for consumers to POST action requests
	// Default is  "/action/digitwin/{agentID}/{thingID}/{key}"
	//  digitwin is addresses the digital twin on the hub
	//  {agentID} is the accountID of the agent under which the TDD was published. This is required.
	//  {thingID} is the digital twin ThingID whose values to read
	//  {key} is the optional key to read. Omit to read all values of the thing.
	// The result is an action status message.
	PostActionPath string `yaml:"postActionPath,omitempty"`

	// PostEventPath is the router path for agents to POST events
	// Default is  "/event/{agentID}/{thingID}/{key}"
	//  {agentID} is the accountID of the agent under which the TDD was published. This is required.
	//  {thingID} is the digital twin ThingID whose values to read
	//  {key} is the optional key to read. Omit to read all values of the thing.
	// The payload is the event content as described by the TDD
	PostEventPath string `yaml:"postEventPath,omitempty"`

	// PostRPCPath is the router path for consumers to POST a RPC request to a service
	// RPC methods are defined as actions in the TDD.
	// This path is included in the TDD action forms.
	//
	// Default is  "/rpc/{serviceID}/{interfaceID}/{methodName}"
	//  {serviceID} is the accountID of the service under which its TDD was published. This is required.
	//  {interfaceID} is the interface to address
	//  {methodName} is the interface method to invoke.
	// The body is the message content as described by the service TDD action.
	// This returns the response payload as described by the service TDD action.
	PostRPCPath string `yaml:"postRpcPath,omitempty"`

	// SSEPath is the router path for sse connections.
	// Agents use this for subscribing to actions and consumers for subscribing to events.
	// Authorization restrictions apply.
	//
	// Default is "/sse"
	// The subscription header must contain a list of subscriptions in the form:
	//    ["/event/{agentID}[/{thingID}[/{key}",...]
	// or
	//    ["/action/{agentID}/{thingID}/{key}",...]
	// The server publishes a json encoded ThingValue object for each event or action.
	SSEPath string `yaml:"ssePath,omitempty"`

	// WSPath is the router path for websocket connections.
	//
	// Agents use this for subscribing to actions and publishing events.
	// Consumers use this for subscribing to events and publishing actions.
	// Authorization restrictions apply.
	//
	// The websocket protocol uses the same message definitions as SSE for subscriptions
	//
	// Default is "/ws"
	// The server publishes a json encoded ThingValue object for each event or action.
	WSPath string `yaml:"wsPath,omitempty"`
}

// NewHttpsBindingConfig creates a new instance of the https binding configuration
// with default values
func NewHttpsBindingConfig() *HttpsBindingConfig {
	cfg := HttpsBindingConfig{
		Host:      "",
		Port:      444,
		EnableSSE: false,
		EnableWS:  false,
		// router paths used in runtime TDD and thing forms
		GetActionsPath: "/action/{agentID}/{thingID}",
		GetTDDPath:     "/thing/directory/{agentID}/{thingID}",
		GetValuesPath:  "/thing/values/{agentID}/{thingID}/{key}",
		PostActionPath: "/action/digitwin/{agentID}/{thingID}/{key}",
		PostEventPath:  "/event/{agentID}/{thingID}/{key}",
		PostRPCPath:    "/rpc/{serviceID}/{interfaceID}/{methodName}",
		SSEPath:        "/sse",
		WSPath:         "/ws",
	}
	return &cfg
}
