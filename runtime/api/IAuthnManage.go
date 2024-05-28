package api

// AuthnManageServiceID is the ThingID of the authentication admin service
const AuthnManageServiceID = "manage"

// admin methods
const (
	// AddAgentMethod requests to add/update a device or service account with auth token
	AddAgentMethod = "addAgent"

	// AddConsumerMethod requests to add a new end-user client
	AddConsumerMethod = "addConsumer"

	// AddServiceMethod requests to add a new service
	AddServiceMethod = "addService"

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

// AddAgentArgs adds a new device agent client.
// Intended for creating an account for IoT device agents.
type AddAgentArgs struct {
	// AgentID of the service or administrator
	AgentID string `json:"clientID"`
	// DisplayName
	DisplayName string `json:"displayName,omitempty"`
	// Public key of agent or "" to generate one
	PubKey string `json:"pubKey,omitempty"`
}
type AddAgentResp struct {
	Token string `json:"token"`
}

// AddConsumerArgs arguments for adding a new client
type AddConsumerArgs struct {
	// ClientID login ID of the end-user
	ClientID string `json:"clientID"`
	// Friendly name of the end-user
	DisplayName string `json:"displayName"`
	// Password authentication
	Password string `json:"password,omitempty"`
}

// AddServiceArgs adds a new service agent account with a keys and token file
// Intended for creating an account for local services or administrators that can
// read the keys and token from the keys directory. Used by the launcher and hub cli.
type AddServiceArgs struct {
	// AgentID that provides services
	AgentID string `json:"agentID"`
	// DisplayName
	DisplayName string `json:"displayName,omitempty"`
	// Public key of agent or "" to generate one
	PubKey string `json:"pubKey,omitempty"`
}
type AddServiceResp struct {
	Token string `json:"token"`
}

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
