package authnstore

import "github.com/hiveot/hub/api/go/authn"

// supported password hashes
const (
	PWHASH_ARGON2id = "argon2id"
	PWHASH_BCRYPT   = "bcrypt" // fallback in case argon2i cannot be used
)

// DefaultPasswordFile is the recommended password filename for Hub authentication
const DefaultPasswordFile = "hub.passwd"

// AuthnEntry containing client profile and password hash
// For internal use.
type AuthnEntry struct {
	// Client's profile
	authn.ClientProfile

	// password encrypted with argon2id or bcrypt
	PasswordHash string
}

// IAuthnStore defined the interface for storing authentication data
type IAuthnStore interface {
	// Add add a device, service or user to the store with authn settings
	//  clientID is the client's identity
	//  profile to add. Empty fields can receive valid defaults.
	Add(clientID string, profile authn.ClientProfile) error

	// Close the store
	Close()

	// Count returns the number of clients in the store
	Count() int

	// Get returns the client's profile
	// Returns an error if the clientID doesn't exist
	Get(clientID string) (profile authn.ClientProfile, err error)

	// List profiles in the store
	List() (entries []authn.ClientProfile, err error)

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

	// Update updates client information
	// If the clientID doesn't exist, this returns an error.
	// This fails if the client doesn't exist.
	Update(clientID string, entry authn.ClientProfile) error

	// VerifyPassword verifies the given password against the stored hash
	// Returns the client profile and an error if the verification fails.
	VerifyPassword(loginID, password string) (authn.ClientProfile, error)
}
