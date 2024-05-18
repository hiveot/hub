package api

// AuthnAdminThingID is the ThingID of the authentication admin service
const AuthnAdminThingID = "authn:admin"

// admin methods
const (
	// AddClientMethod requests to add a new client
	AddClientMethod = "addClient"

	// GetClientProfileMethod requests the profile of a client
	GetClientProfileMethod = "getClientProfile"

	// GetProfilesMethod requests a list of all clients, users, services and agents
	GetProfilesMethod = "getProfiles"

	// NewAuthTokenMethod requests a new authentication token for agent or service
	NewAuthTokenMethod = "newAuthToken"

	// RemoveClientMethod requests removal of a client
	RemoveClientMethod = "removeClient"

	// SetClientPasswordMethod requests changing a client's password
	SetClientPasswordMethod = "setClientPassword"

	// UpdateClientProfileMethod requests updates to a client
	UpdateClientProfileMethod = "updateClientProfile"
)

// AuthnEntry containing client profile and password hash
// For internal use.
type AuthnEntry struct {
	// Client's profile
	ClientProfile `yaml:"clientProfile" json:"clientProfile"`

	// PasswordHash password encrypted with argon2id or bcrypt
	PasswordHash string `yaml:"passwordHash" json:"passwordHash"`

	// Client 'base role'. Authz can add agent/thing specific roles in the future.
	// This is set when creating a user and updated with SetRole. Authz reads it.
	Role string `yaml:"role" json:"role"`
}

// AddClientArgs arguments for adding a new client
type AddClientArgs struct {
	ClientType  ClientType `json:"clientType"`
	ClientID    string     `json:"clientID"`
	DisplayName string     `json:"displayName"`
	PubKey      string     `json:"pubKey"`
	Password    string     `json:"password"`
}

// GetClientProfileArgs arguments for requesting a client's profile
type GetClientProfileArgs struct {
	ClientID string `json:"clientID"`
}

// GetClientProfileResp response with a client's profile.
type GetClientProfileResp struct {
	Profile ClientProfile `json:"profile"`
}

// GetProfilesResp response message to get the a list of client profiles.
type GetProfilesResp struct {
	Profiles []ClientProfile `json:"profiles"`
}

// NewAuthTokenArgs creates an auth token for an agent or service
// Intended for services and agents
type NewAuthTokenArgs struct {
	// ClientID of the service or agent
	ClientID string `json:"clientID"`
	// Optional duration of the token, or empty for the default duration
	ValiditySec int `json:"validity,omitempty"`
}

// NewAuthTokenResp contains the new token
type NewAuthTokenResp struct {
	Token string `json:"token"`
}

// RemoveClientArgs arguments for removing a client from the system
type RemoveClientArgs struct {
	ClientID string `json:"clientID"`
}

// SetClientPasswordArgs arguments for updating a client's password
type SetClientPasswordArgs struct {
	ClientID string `json:"clientID"`
	Password string `json:"password"`
}

// UpdateClientProfileArgs arguments for updating a client's profile
type UpdateClientProfileArgs struct {
	Profile ClientProfile `json:"profile"`
}
