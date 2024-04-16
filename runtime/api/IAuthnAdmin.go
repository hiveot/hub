package api

// AuthnAdminServiceID is the ThingID of the admin service
const AuthnAdminServiceID = "authnAdmin"

// admin methods
const (
	// AddClientMethod requests to add a new client
	AddClientMethod = "addClient"

	// GetClientProfileMethod requests the profile of a client
	GetClientProfileMethod = "getClientProfile"

	// GetProfilesMethod requests a list of all clients, users, services and agents
	GetProfilesMethod = "getProfiles"

	// RemoveClientMethod requests removal of a client
	RemoveClientMethod = "removeClient"

	// UpdateClientProfileMethod requests updates to a client
	UpdateClientProfileMethod = "updateClientProfile"

	// UpdateClientPasswordMethod requests updates to a client's password
	UpdateClientPasswordMethod = "updateClientPassword"
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

// RemoveClientArgs arguments for removing a client from the system
type RemoveClientArgs struct {
	ClientID string `json:"clientID"`
}

// UpdateClientPasswordArgs arguments for updating a client's password
type UpdateClientPasswordArgs struct {
	ClientID string `json:"clientID"`
	Password string `json:"password"`
}

// UpdateClientProfileArgs arguments for updating a client's profile
type UpdateClientProfileArgs struct {
	Profile ClientProfile `json:"profile"`
}
