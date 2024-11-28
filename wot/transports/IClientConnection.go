// Package transports with the interface of a client transport connection
package transports

import "github.com/hiveot/hub/wot/tdd"

// Supported transport protocol bindings types
const (
	ProtocolTypeHTTP  = "https"
	ProtocolTypeSSE   = "sse"   // subprotocol of https
	ProtocolTypeSSESC = "ssesc" // subprotocol of https
	ProtocolTypeWSS   = "wss"   // subprotocol of https
	ProtocolTypeMQTT  = "mqtt"
)

// MessageHandler processes a message without expecting a return value
type MessageHandler func(msg *ThingMessage)

// RequestHandler processes a request and returns a progress status.
//
//	msg is the envelope that contains the request to process
type RequestHandler func(msg *ThingMessage) RequestStatus

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

// IClientConnection defines the client interface of a transport protocol binding
type IClientConnection interface {
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

	// GetProtocolType returns the (sub)protocol type implemented by this client
	// See the  ProtocolType constants defined above.
	// See also: https://www.w3.org/TR/wot-binding-templates/#protocol-bindings-table
	GetProtocolType() string

	// GetServerURL returns the schema://address:port of the server
	GetServerURL() string

	// IsConnected returns true when the connection is active
	IsConnected() bool

	// Logout of the hub and invalidate its session.
	// This will disconnect all existing connections from the hub and invalidate
	// all authentication tokens.
	// The client has to use login with password to reauthenticate.
	Logout() error

	// RefreshToken refreshes the authentication token
	// The resulting token can be used with 'ConnectWithToken'
	RefreshToken(oldToken string) (newToken string, err error)

	// Rpc sends an operation and waits for a status response or until timeout.
	// It is similar to SendOperation but waits until the response is either
	// StatusCompleted, StatusFailed, or until a timeout occurs.
	// If the operation has no result and no response status is expected then
	// use SendOperation instead.
	Rpc(form tdd.Form, dThingID, name string, input interface{},
		output interface{}) error

	// SendOperation sends the operation described in the given Form.
	// This can be any of the predefined operations that are supported on the server.
	//
	// form is the Thing TD form that describes the transport parameters needed by this
	// protocol binding. It is provided by the Thing's TD.
	// dThingID and name are injected in the form href, if used. (uri variables)
	// input contains the input data as per affordance.
	// output is the operation's output
	//
	// This returns a progress status value object (pending, failed or completed),
	// or an error if the operation failed to complete, including timeout.
	SendOperation(form tdd.Form, dThingID, name string, input interface{},
		output interface{}, correlationID string) (status string, err error)

	// SendOperationStatus [agent] sends a operation progress status update
	// to the server.
	//
	// Intended for agents that have processed an incoming action request
	// asynchronously and need to send an update on further progress.
	SendOperationStatus(stat RequestStatus)

	// SetMessageHandler sets the handler for event style operations received from the server.
	// This replaces any previously set handler.
	SetMessageHandler(cb MessageHandler)

	// SetRequestHandler sets the handler for operations that return a response.
	// This replaces any previously set handler.
	SetRequestHandler(cb RequestHandler)

	// SetConnectHandler sets the notification handler of connection status changes
	SetConnectHandler(cb func(connected bool, err error))
}
