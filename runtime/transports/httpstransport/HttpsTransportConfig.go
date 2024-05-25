package httpstransport

const DefaultHttpsPort = 8444

// HttpsTransportConfig contains the configuration of the HTTPS server binding,
// including the websocket and the sse configuration.
type HttpsTransportConfig struct {
	// enable websocket support. Default is false.
	EnableWS bool `yaml:"enableWS,omitempty"`

	// enable SSE support. Default is false.
	EnableSSE bool `yaml:"enableSSE,omitempty"`

	// Host is the server address, default is outbound IP address
	Host string `yaml:"host,omitempty"`

	// Port is the server TLS port, default is 8444
	// This port handles http, websocket and sse requests
	Port int `yaml:"port,omitempty"`
}

// NewHttpsTransportConfig creates a new instance of the https binding configuration
// with default values
func NewHttpsTransportConfig() HttpsTransportConfig {
	cfg := HttpsTransportConfig{
		Host:      "",
		Port:      DefaultHttpsPort,
		EnableSSE: false,
		EnableWS:  false,
	}
	return cfg
}
