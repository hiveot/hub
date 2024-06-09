package api

import "github.com/hiveot/hub/api/go/authn"

// AuthnEntry containing client profile and password hash
// For internal use.
type AuthnEntry struct {
	// Client's profile
	authn.ClientProfile `yaml:"clientProfile" json:"clientProfile"`

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
	Add(clientID string, profile authn.ClientProfile) error

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
	GetProfile(clientID string) (profile authn.ClientProfile, err error)

	// GetProfiles returns all client profiles in the store
	GetProfiles() (entries []authn.ClientProfile, err error)

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
	UpdateProfile(clientID string, profile authn.ClientProfile) error

	// VerifyPassword verifies the given password against the stored hash
	// Returns the client profile and an error if the verification fails.
	VerifyPassword(loginID, password string) (authn.ClientProfile, error)
}
