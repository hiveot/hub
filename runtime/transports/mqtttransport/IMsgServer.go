package mqtttransport

import "github.com/hiveot/hub/lib/keys"

// ClientAuthInfo defines client authentication and authorization information
type ClientAuthInfo struct {
	// UserID, AgentID or AgentID of the client
	ClientID string

	// ClientType identifies the client as a ClientTypeDevice, ClientTypeService or ClientTypeUser
	ClientType string

	// The PEM encoded client's public key, if any
	PubKey string

	// password encrypted with argon2id or bcrypt
	PasswordHash string

	// The client's role
	Role string // Name of user's role
}

// RolePermission defines authorization for a role.
// Each permission defines the source/things the user can pub/sub to.
type RolePermission struct {
	AgentID  string // device or service publishing the Thing data, or "" for all
	ThingID  string // thingID or capability, or "" for all
	MsgType  string // rpc, event, action, config, or "" for all message types
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

	// ConnectInProc creates an in-process client connection to the server using a generated token for the serviceID.
	//
	// Optionally provide an alternative key-pair, or use nil for the predefined core service key.
	// the provided keypair is that of a server generated keypair. See CreateKeys()
	//ConnectInProc(serviceID string) (hubclient.IHubClient, error)

	// Core returns the running core, "nats" or "mqtt"
	Core() string

	// CreateKeyPair creates a serialized keypair for use in connecting or signing.
	// This returns the crypto key pair,
	// NOTE: intended for testing. Might be deprecated in the future.
	CreateKeyPair() (kp keys.IHiveKey)

	// CreateToken creates a new authentication token for a known client.
	// The client must have been added with ApplyAuth and have a public key.
	//
	// The type of token created depends on the server configuration.
	//  NATS nkey server simply returns the public key for connecting with nkey.
	//  NATS callout server returns a JWT token containing authorization.
	//  MQTT uses a base64 encoded JWT token.
	//
	//  authInfo with client info used to create and verify the token
	CreateToken(authInfo ClientAuthInfo) (token string, err error)

	// GetServerURLs returns the server URLs
	GetServerURLs() (tlsURL string, wssURL string, udsURL string)

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
	// Use GetServerURLS to determine the supported connection addresses
	Start() error

	// Stop the server
	Stop()

	// ValidatePassword verifies the password for the user using the ApplyAuth users
	//  loginID is the client ID of the user
	//  password is the bcrypt encoded password???
	// On success this returns the client auth information useful for creating tokens
	//ValidatePassword(loginID string, password string) (ClientAuthInfo,error)

	// ValidateToken verifies whether the given authentication token is valid
	//
	// Validation depends on the server implementation.
	// The use of nonce in signing and verification is optional but recommended. It depends
	// on availability of the underlying messaging system.
	//
	//  clientID to whom the token is issued
	//  token to verify
	//  signedNonce base64 encoded signature generated from private key and nonce field
	//  nonce the server provided field used to sign the token.
	ValidateToken(clientID string, token string, signedNonce string, nonce string) error
}
