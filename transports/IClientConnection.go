// Package transports with the interface of a client transport connection
package transports

import (
	"github.com/hiveot/hub/wot/td"
)

// GetFormHandler is the handler that provides the client with the form needed to invoke an operation
// This returns nil if no form is found for the operation.
type GetFormHandler func(op string, thingID string, name string) *td.Form

// IClientConnection defines the client interface for establishing connections with a server
// Intended for consumers to connect to a Thing Agent/Hub and for Service agents that connect
// to the Hub.
type IClientConnection interface {
	IConnection

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
	// If a connection is already established on this client then it will be closed first.
	//
	// This connection method must be supported by all transport implementations.
	ConnectWithToken(token string) (err error)
}
