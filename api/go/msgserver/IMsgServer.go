package msgserver

import (
	"github.com/hiveot/hub/api/go/hubclient"
)

// ClientAuthInfo defines client authentication and authorization information
type ClientAuthInfo struct {
	// UserID, ServiceID or DeviceID of the client
	ClientID string

	// ClientType identifies the client as a ClientTypeDevice, ClientTypeService or ClientTypeUser
	ClientType string

	// The client's public key, if any
	PubKey string

	// password encrypted with argon2id or bcrypt
	PasswordHash string

	// The client's role
	Role string // Name of user's role
}

// RolePermission defines authorization for a role.
// Each permission defines the source/thing the user can pub/sub to.
type RolePermission struct {
	Prefix   string // things (default) or "svc" for service
	SourceID string // source or "" for all sources
	ThingID  string // thingID or "" for all things
	MsgType  string // event or action, or "" for all message types
	MsgName  string // action name or "" for all actions
	AllowPub bool   // allow publishing of this message
	AllowSub bool   // allow subscribing to this message
}

// IMsgServer defines the interface of the messaging server
type IMsgServer interface {
	// ApplyAuth applies authentication configuration to the server config.
	// As messaging servers have widely different ways of handling authentication and
	// authorization this simply gives all users and roles to the server to apply
	// as it sees fit. The server implements the server specific portion.
	//
	//  clients is the list of registered users and sources with their credentials
	ApplyAuth(clients []ClientAuthInfo) error

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
	//  clientID of a known client
	CreateToken(clientID string) (newToken string, err error)

	// SetRolePermissions sets the roles used in authorization.
	// As messaging servers have widely different ways of handling authentication and
	// authorization this simply gives all users and roles to the server to apply
	// as it sees fit. The server implements the server specific portion.
	//
	//  rolePerm is a map of [role]permissions. Use nil to revert back to the default role permissions.
	SetRolePermissions(rolePerm map[string][]RolePermission)

	// SetServicePermissions sets the roles that are allowed to use a service capability.
	// This amends the role permissions with the service capabilities.
	// Intended for registering services.
	SetServicePermissions(serviceID string, capability string, roles []string)

	// Start the server.
	// This returns the primary connection address for use in discovery.
	Start() (clientURL string, err error)

	// Stop the server
	Stop()

	// ValidatePassword verifies the password for the user using the ApplyAuth users
	//  loginID is the client ID of the user
	//  password is the bcrypt encoded password???
	ValidatePassword(loginID string, password string) error

	// ValidateToken verifies whether the token is valid
	// The token must contain the public key of the client for verification.
	// NATS uses the 'sub' field for its public key. The provided public key on record can
	// be used as an extra verification step.
	// The use of nonce in signing and verification is optional but recommended. It depends
	// on availability of the underlying messaging system.
	//
	//  clientID to whom the token is issued
	//  pubKey of the client for extra verification
	//  token to verify
	//  signedNonce base64 encoded signature generated from private key and nonce field
	//  nonce the server provided field used to sign the token.
	ValidateToken(clientID string, pubKey string, token string, signedNonce string, nonce string) error
}
