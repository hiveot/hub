package authn

// ClientType with the types of clients supported by authn
type ClientType string

const (
	ClientTypeAgent   ClientType = "agent"
	ClientTypeService ClientType = "service"
	ClientTypeUser    ClientType = "user"
)

// Session token validity for client types
const (
	DefaultAgentTokenValiditySec   = 90 * 24 * 3600  // 90 days
	DefaultServiceTokenValiditySec = 365 * 24 * 3600 // 1 year
	DefaultUserTokenValiditySec    = 30 * 24 * 3600  // 30 days
)

// supported password hashes
const (
	PWHASH_ARGON2id = "argon2id"
	PWHASH_BCRYPT   = "bcrypt" // fallback in case argon2id cannot be used
)

// DefaultAdminUserID is the client ID of the default CLI administrator account
const DefaultAdminUserID = "admin"

// DefaultLauncherServiceID is the client ID of the launcher service
// auth creates a key and auth token for the launcher on startup
const DefaultLauncherServiceID = "launcher"

// DefaultPasswordFile is the recommended password filename for Hub authentication
const DefaultPasswordFile = "hub.passwd"

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
	UpdatedMSE int64 `json:"updatedMSE,omitempty"`
	// TokenValidityDays nr of seconds that issued JWT tokens are valid for or 0 for default
	TokenValiditySec int `json:"tokenValiditySec,omitempty"`
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

// IAuthnStore defined the interface for storing authentication data
type IAuthnStore interface {
	// Add adds a device, service or user to the store with authn settings
	// If the client already exists, it is updated with the profile.
	//
	//  clientID is the client's identity
	//  profile to add. Empty fields can receive valid defaults.
	Add(clientID string, profile ClientProfile) error

	// Close the store
	Close()

	// Count returns the number of clients in the store
	Count() int

	// GetEntries returns a list of client profiles including the password hash
	// Intended to obtain auth info to apply to the messaging server
	// For internal auth usage only.
	GetEntries() (entries []AuthnEntry)

	// GetProfile returns the client's profile
	// Returns an error if the clientID doesn't exist
	GetProfile(clientID string) (profile ClientProfile, err error)

	// GetProfiles returns all client profiles in the store
	GetProfiles() (entries []ClientProfile, err error)

	// GetRole returns the client's default role
	GetRole(clientID string) (role string, err error)

	// Open the store
	Open() error

	// Remove the client from the store
	// If the client doesn't exist, no error is returned
	Remove(clientID string) (err error)

	// SetPassword stores the hash of the password for the given user.
	// If the clientID doesn't exist, this returns an error.
	//
	// The hashing algorithm is embedded in the store.
	//  clientID is the login ID of the user whose hash to write
	//  password is the password whose hash to store
	// Returns error if the store isn't writable
	SetPassword(clientID string, password string) error

	// SetRole sets the default role of a client
	SetRole(clientID string, newRole string) error

	// UpdateProfile updates client profile
	// If the clientID doesn't exist, this returns an error.
	// This fails if the client doesn't exist.
	UpdateProfile(clientID string, entry ClientProfile) error

	// VerifyPassword verifies the given password against the stored hash
	// Returns the client profile and an error if the verification fails.
	VerifyPassword(loginID, password string) (ClientProfile, error)
}
