// Package protocolclients with the interface of a client transport connection
package transports

import (
	"github.com/hiveot/hub/wot/tdd"
)

// Supported transport protocol bindings types
const (
	ProtocolTypeHTTPS = "https"
	ProtocolTypeSSE   = "sse"   // subprotocol of https
	ProtocolTypeSSESC = "ssesc" // subprotocol of https
	ProtocolTypeWSS   = "wss"   // subprotocol of https
	ProtocolTypeMQTT  = "mqtt"
)

// TransportStatus connection status of a hub client transport
//type TransportStatus struct {
//	// URL of the hub
//	HubURL string
//	// CA used to connect
//	CaCert *x509.Certificate
//	// the client ID to identify as
//	//SenderID string
//
//	// The current connection status
//	isConnected bool
//
//	// The last connection error message, if any
//	LastError error
//
//	// flags indicating the supported protocols
//	SupportsCertAuth     bool
//	SupportsPasswordAuth bool
//	SupportsKeysAuth     bool
//	SupportsTokenAuth    bool
//}

// IProtocolBindingClient defines the client interface of a protocol binding
type IProtocolBindingClient interface {
	// ConnectWithClientCert connects to the server using a client certificate.
	// This authentication method is optional
	//ConnectWithClientCert(kp keys.IHiveKey, cert *tls.Certificate) (err error)

	// ConnectWithPassword connects to the messaging server using password authentication.
	// If a connection already exists it will be closed first.
	//
	// This returns a connection token that can be used with ConnectWithToken.
	//
	//  password is created when registering the user with the auth service.
	//
	// This authentication method must be supported by all transport implementations.
	ConnectWithPassword(password string) (newToken string, err error)

	// ConnectWithToken connects to the messaging server using an authentication token.
	//
	// If a connection already exists it will be closed first.
	//
	// and pub/private keys provided when creating an instance of the hub client.
	//  token is created by the auth service.
	//
	// This authentication method must be supported by all transport implementations.
	ConnectWithToken(token string) (newToken string, err error)

	// CreateKeyPair returns a new set of serialized public/private key pair.
	//  serializedKP contains the serialized public/private key pair
	//  pubKey contains the serialized public key to be shared
	//CreateKeyPair() (kp keys.IHiveKey)

	// Disconnect from the messaging server.
	// This retains the session and allows a reconnect with the session token.
	// See Logout for invalidating existing sessions
	Disconnect()

	// GetClientID returns the agent or user clientID for this hub client
	GetClientID() string

	// GetConnectionStatus returns the current two-way connection status
	// If disconnected, the last error is included.
	GetConnectionStatus() (connected bool, cid string, err error)

	// GetProtocolType returns the (sub)protocol type supported by this client
	// See the  ProtocolType constants defined above.
	// See also: https://www.w3.org/TR/wot-binding-templates/#protocol-bindings-table
	GetProtocolType() string

	// GetServerURL returns the schema://address:port of the server
	GetServerURL() string

	// InvokeOperation invokes the operation from a Form
	// dThingID and name are injected in the form href, if used.
	// input contains the input data as per affordance
	// output is the optional result in case of actions or read requests
	InvokeOperation(op tdd.Form, dThingID, name string, input interface{}, output interface{}) error

	// Logout of the hub and invalidate the connected session token.
	// This will disconnect from the hub.
	Logout() error

	// RefreshToken refreshes the authentication token
	// The resulting token can be used with 'ConnectWithToken'
	RefreshToken(oldToken string) (newToken string, err error)

	// SetMessageHandler adds a handler for messages from the hub.
	// This replaces any previously set handler.
	//
	// To receive events use the 'Subscribe' method to set the events to listen for.
	// To receive property updates use 'Observe'.
	// For agents to receive actions, no subscription is necessary.
	SetMessageHandler(cb MessageHandler)

	// SetConnectHandler sets the notification handler of connection status changes
	SetConnectHandler(cb func(connected bool, err error))
}
