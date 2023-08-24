package msgserver

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/api/go/hubclient"
)

// IMsgServer defines the interface of the messaging server
type IMsgServer interface {
	// ApplyAuthn applies updated authentication configuration to the server config.
	// As messaging servers have widely different ways of handling authentication and
	// authorization this simply gives all users and groups to the server to apply
	// as it sees fit.
	// The server implements the server specific portion. This is intended for use
	// by core services to apply configuration changes.
	ApplyAuthn(clients []authn.AuthnEntry) error

	// ApplyAuthz applies updated authorization configuration to the server config.
	// As messaging servers have widely different ways of handling authentication and
	// authorization this simply gives all users and groups to the server to apply
	// as it sees fit.
	// The server implements the server specific portion. This is intended for use
	// by core services to apply configuration changes.
	ApplyAuthz(userGroupRoles map[string]authz.RoleMap) error

	// ApplyGroups is invoked after changes to groups
	// The server synchronizes its groups with the given list
	ApplyGroups(groups []authz.Group) error

	// ConnectInProc creates an in-process client connection to the server.
	//
	// Optionally provide an alternative key-pair, or use nil for the predefined core service key.
	// the provided keypair is that of a server generated keypair. See CreateKeys()
	ConnectInProc(serviceID string) (hubclient.IHubClient, error)

	// CreateKP creates a keypair for use in connecting or signing. This can be used
	// with ConnectInProc.
	// This returns the key pair and public key string.
	//CreateKP() (interface{}, string)

	// CreateToken creates a new authentication token that can be used to connect.
	// The type of token created depends on the server configuration.
	//  NATS nkey server simply returns the public key for connecting with nkey
	//  NATS callout server returns a JWT token for connecting with JWT
	//
	//  clientID with the identity of the device, service or user
	//  clientType ClientTypeDevice, ClientTypeService or ClientTypeUser
	//  pubKey public key string to use with the token
	//  validitySec with the lifespan in seconds (if supported)
	CreateToken(clientID string, clientType string, pubKey string, validitySec int) (newToken string, err error)

	// Start the server.
	// This returns the primary connection address for use in discovery.
	Start() (clientURL string, err error)

	// Stop the server
	Stop()

	// ValidateToken verifies whether the token is valid
	// The token must contain the public key of the client for verification.
	// NATS uses the 'sub' field for example. The provided public key on record can
	// be used as an extra verification step.
	// The use of nonce in signing and verification is optional but recommended. It depends
	// on availability of the underlying messaging system.
	//
	//  clientID to whom the token is issued
	//  token to verify
	//  signedNonce base64 encoded signature generated from private key and nonce field
	//  nonce the server provided field used to sign the token.
	ValidateToken(clientID string, pubKey string, token string, signedNonce string, nonce string) error
}
