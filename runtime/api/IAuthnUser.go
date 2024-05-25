package api

// AuthnAgentID is the connection ID of the agent used in providing its capabilities
const AuthnAgentID = "authn"

// AuthnUserServiceID contains the ThingID of the authentication user service
const AuthnUserServiceID = "authnuse"

// client methods
const (
	// GetProfileMethod requests the profile of the requesting client
	GetProfileMethod = "getProfile"

	// LoginMethod requests an authentication token using password based login
	LoginMethod = "login"

	// RefreshTokenMethod requests a new token for the current client
	//
	// This returns a new short-lived auth token that can be used to authenticate with the hub
	// This requires the client's public key on file.
	RefreshTokenMethod = "refresh"

	// UpdateNameMethod requests changing the display name of the current client
	UpdateNameMethod = "updateName"

	// UpdatePasswordMethod requests changing the password of the current client
	UpdatePasswordMethod = "updatePassword"

	// UpdatePubKeyMethod requests changing the public key on file of the current client.
	// The public key is used in token validation and generation.
	// This takes effect immediately. Existing connection must be closed and re-established.
	UpdatePubKeyMethod = "updatePubKey"
)

// ClientType with the types of clients supported by authn
type ClientType string

const (
	// ClientTypeAgent is used for clients that are IoT devices.
	ClientTypeAgent ClientType = "agent"
	// ClientTypeService is used for agents that provide a service. Services have
	// additional permissions to publish actions and receive events.
	ClientTypeService ClientType = "service"
	// ClientTypeUser is used for end-users. End user auth tokens are restricted to a session.
	ClientTypeUser ClientType = "user"
)

// ClientProfile contains client information of sources and users
type ClientProfile struct {
	// The client ID.
	//  for users this is their email
	//  for IoT devices or services, use the bindingID
	//  for services the service instance ID
	ClientID string `json:"clientID,omitempty"`
	// ClientType identifies the client as a ClientTypeDevice, ClientTypeService or ClientTypeUser
	ClientType ClientType `json:"clientType,omitempty"`
	// The client presentation name
	DisplayName string `json:"displayName,omitempty"`
	// The client's PEM encoded public key
	PubKey string `json:"pubKey,omitempty"`
	// timestamp in 'Millisec-Since-Epoc' the entry was last updated
	UpdatedMsec int64 `json:"updatedMsec,omitempty"`
	// TokenValidityDays nr of seconds that issued JWT tokens are valid for or 0 for default
	TokenValiditySec int `json:"tokenValiditySec,omitempty"`
}

// LoginArgs request arguments for password based login to the hub
// FIXME: instead of cleartext password use a hash based on the server provided nonce
type LoginArgs struct {
	// ID of the user logging in
	ClientID string `json:"clientID"`
	// Password
	Password string `json:"password"`
	// Client nonce
	//Nonce string `json:"nonce"`
}

// LoginResp contains the authentication token if login is successful
type LoginResp struct {
	Token string `json:"token"`
}

// GetProfileResp response message to get the current client's profile.
type GetProfileResp struct {
	Profile ClientProfile `json:"profile"`
}

// RefreshTokenArgs arguments for requesting a new user login token
type RefreshTokenArgs struct {
	ClientID string `json:"clientID"`
	OldToken string `json:"oldToken"`
}

// RefreshTokenResp contains the new token
type RefreshTokenResp struct {
	Token string `json:"token"`
}

// UpdateNameArgs arguments for requesting a user name update
type UpdateNameArgs struct {
	NewName string `json:"newName"`
}

// UpdatePasswordArgs arguments for requesting a user password update
type UpdatePasswordArgs struct {
	NewPassword string `json:"newPassword"`
}

// UpdatePubKeyArgs arguments for updating a public key in pem format
type UpdatePubKeyArgs struct {
	PubKeyPem string `json:"pubKeyPem"`
}
