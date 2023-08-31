package auth

// AuthManageProfileCapability is the name of the Thing/Capability that handles client requests
const AuthManageProfileCapability = "manageProfile"

// below a list of actions and their payload

// GetProfileAction defines the action to get the client's profile
const GetProfileAction = "getProfile"

// GetProfileReq request message to get a client's profile.
type GetProfileReq struct {
	// must match the login ID
	ClientID string `json:"clientID"`
}
type GetProfileResp struct {
	Profile ClientProfile `json:"profile"`
}

// NewTokenAction requests a new jwt token for password based login
const NewTokenAction = "newToken"

type NewTokenReq struct {
	// ClientID is the authentication ID of the client publishing the action
	ClientID string `json:"clientID"`
	// Password to verify identity
	Password string `json:"password"`
}
type NewTokenResp struct {
	Token string `json:"Token"`
}

const RefreshAction = "refresh"

type RefreshReq struct {
	ClientID string `json:"clientID"`
	// old token must match clientID
	OldToken string `json:"oldToken"`
}
type RefreshResp struct {
	NewToken string `json:"newToken"`
}

const UpdateNameAction = "updateName"

type UpdateNameReq struct {
	// ClientID to update. For regular users this must match the loginID.
	ClientID string `json:"clientID"`
	NewName  string `json:"newName"`
}

const UpdatePasswordAction = "updatePassword"

type UpdatePasswordReq struct {
	ClientID    string `json:"clientID"`
	NewPassword string `json:"newPassword"`
}

const UpdatePubKeyAction = "updatePubKey"

type UpdatePubKeyReq struct {
	ClientID  string `json:"clientID"`
	NewPubKey string `json:"newPubKey"`
}

// IAuthManageProfile defines the auth capability for use by hub clients.
// Regular clients have permissions to manage their profile and get new auth tokens.
type IAuthManageProfile interface {

	// GetProfile returns a client's profile
	GetProfile(clientID string) (profile ClientProfile, err error)

	// NewToken validates a password and returns a new auth token.
	// This returns a short-lived auth token that can be used to connect to the message server
	// The token can be refreshed to extend it without requiring a login password.
	// A public key must be on file for this to work.
	NewToken(clientID string, password string) (jwtToken string, err error)

	// Refresh a short-lived authentication token.
	//
	//  oldToken must be a valid token obtained at login or refresh
	//
	// This returns a new short-lived auth token that can be used to authenticate with the hub
	// This fails if the token has expired or does not belong to the clientID
	Refresh(clientID string, oldToken string) (JwtToken string, err error)

	// UpdateName updates a user's display name
	UpdateName(clientID string, newName string) (err error)

	// UpdatePassword changes the client password
	// This requires a valid login as the client.
	UpdatePassword(clientID string, newPassword string) error

	// UpdatePubKey changes the public key on file
	// This requires a valid login as the client.
	// This takes effect immediately. Existing connection must be closed and re-established.
	UpdatePubKey(clientID string, newPubKey string) error
}
