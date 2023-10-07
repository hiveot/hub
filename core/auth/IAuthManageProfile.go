package auth

// AuthProfileCapability is the name of the Thing/Capability that handles client requests
const AuthProfileCapability = "profile"

// below a list of actions and their payload

// GetProfileReq defines the request to get the current client's profile
const GetProfileReq = "getProfile"

// GetProfileResp response message to get the client's profile.
// The message address MUST contain the client sending the action to whom this applies
type GetProfileResp struct {
	Profile ClientProfile `json:"profile"`
}

// NewTokenReq requests a new jwt token for password based login
// The message address MUST contain the client sending the action to whom this applies
const NewTokenReq = "newToken"

type NewTokenArgs struct {
	// Password to verify identity
	Password string `json:"password"`
}
type NewTokenResp struct {
	Token string `json:"Token"`
}

// RefreshTokenReq requests a new token using an existing token for the current client
const RefreshTokenReq = "refresh"

type RefreshResp struct {
	NewToken string `json:"newToken"`
}

// UpdateNameReq requests changing the display name of the current client
const UpdateNameReq = "updateName"

type UpdateNameArgs struct {
	NewName string `json:"newName"`
}

// UpdatePasswordReq requests changing the password of the current client
const UpdatePasswordReq = "updatePassword"

type UpdatePasswordArgs struct {
	NewPassword string `json:"newPassword"`
}

// UpdatePubKeyReq requests changing the public key on file of the current client
// The public key is used in token validation and generation.
const UpdatePubKeyReq = "updatePubKey"

type UpdatePubKeyArgs struct {
	NewPubKey string `json:"newPubKey"`
}

// IAuthManageProfile defines the auth capability for use by hub clients.
// Regular clients have permissions to manage their profile and get new auth tokens.
type IAuthManageProfile interface {

	// GetProfile returns the connected client's profile
	GetProfile() (profile ClientProfile, err error)

	// NewToken validates a password and returns a new auth token.
	// This returns a short-lived auth token that can be used to connect to the message server
	// The token can be refreshed to extend it without requiring a login password.
	// A public key must be on file for this to work.
	NewToken(password string) (jwtToken string, err error)

	// Refresh a short-lived authentication token.
	//
	//  oldToken must be a valid token obtained at login or refresh
	//
	// This returns a new short-lived auth token that can be used to authenticate with the hub
	// This fails if the token has expired or does not belong to the clientID
	Refresh() (JwtToken string, err error)

	// UpdateName updates a user's display name
	UpdateName(newName string) (err error)

	// UpdatePassword changes the client password
	UpdatePassword(newPassword string) error

	// UpdatePubKey changes the public key on file
	// This takes effect immediately. Existing connection must be closed and re-established.
	UpdatePubKey(newPubKey string) error
}
