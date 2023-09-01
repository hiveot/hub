package auth

import "time"

// IAuthnTokenizer is the interface of the token generator and validator for the
// underlying messaging authentication.
type IAuthnTokenizer interface {
	// CreateToken creates a new authentication token that is accepted by the
	// underlying messaging authentication.
	// This can implement rules in the token that restrict publication and subscriptions.
	//
	//  clientID with the identity of the device, service or user
	//  clientType ClientTypeDevice, ClientTypeService or ClientTypeUser
	//  pubKey public key string to include in the token
	//  validitySec with the token lifespan
	CreateToken(clientID string, clientType string, pubKey string, validity time.Duration) (newToken string, err error)

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
