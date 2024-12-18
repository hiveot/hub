package httptransport

const DefaultHttpsPort = 8444

// HttpTransportConfig contains the configuration of the HTTPS server binding,
// including the websocket and the sse configuration.
type HttpTransportConfig struct {
	// enable websocket support.
	EnableWS bool `yaml:"enableWS,omitempty"`

	// enable SSE support.
	EnableSSE bool `yaml:"enableSSE,omitempty"`

	// enable SSE-SC support.
	EnableSSESC bool `yaml:"enableSSESC,omitempty"`

	// Host is the server address, default is outbound IP address
	Host string `yaml:"host,omitempty"`

	// Port is the server TLS port, default is 8444
	// This port handles http, websocket and sse requests
	Port int `yaml:"port,omitempty"`

	// token validity when logging in using REST
	// Default is DefaultConsumerTokenValiditySec
	//ConsumerTokenValiditySec int `yaml:"consumerTokenValiditySec"`
}

// NewHttpTransportConfig creates a new instance of the https binding configuration
// with default values
func NewHttpTransportConfig() HttpTransportConfig {
	cfg := HttpTransportConfig{
		Host:        "",
		Port:        DefaultHttpsPort,
		EnableSSE:   false,
		EnableSSESC: true,
		EnableWS:    false,
	}
	return cfg
}
