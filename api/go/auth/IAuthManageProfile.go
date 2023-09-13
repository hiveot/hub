package auth

// AuthProfileCapability is the name of the Thing/Capability that handles client requests
const AuthProfileCapability = "profile"

// below a list of actions and their payload

// GetProfileAction defines the action to get the client's profile
const GetProfileAction = "getProfile"

// GetProfileResp response message to get the client's profile.
// The message address MUST contain the client sending the action to whom this applies
type GetProfileResp struct {
	Profile ClientProfile `json:"profile"`
}

// NewTokenAction requests a new jwt token for password based login
// The message address MUST contain the client sending the action to whom this applies
const NewTokenAction = "newToken"

type NewTokenReq struct {
	// Password to verify identity
	Password string `json:"password"`
}
type NewTokenResp struct {
	Token string `json:"Token"`
}

// The message address MUST contain the client sending the action to whom this applies
const RefreshAction = "refresh"

type RefreshReq struct {
	// old token must match clientID
	OldToken string `json:"oldToken"`
}
type RefreshResp struct {
	NewToken string `json:"newToken"`
}

// The message address MUST contain the client sending the action to whom this applies
const UpdateNameAction = "updateName"

type UpdateNameReq struct {
	NewName string `json:"newName"`
}

// The message address MUST contain the client sending the action to whom this applies
const UpdatePasswordAction = "updatePassword"

type UpdatePasswordReq struct {
	NewPassword string `json:"newPassword"`
}

// The message address MUST contain the client sending the action to whom this applies
const UpdatePubKeyAction = "updatePubKey"

type UpdatePubKeyReq struct {
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
	Refresh(oldToken string) (JwtToken string, err error)

	// UpdateName updates a user's display name
	UpdateName(newName string) (err error)

	// UpdatePassword changes the client password
	UpdatePassword(newPassword string) error

	// UpdatePubKey changes the public key on file
	// This takes effect immediately. Existing connection must be closed and re-established.
	UpdatePubKey(newPubKey string) error
}
