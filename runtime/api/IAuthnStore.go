package api

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
	UpdateProfile(clientID string, profile ClientProfile) error

	// VerifyPassword verifies the given password against the stored hash
	// Returns the client profile and an error if the verification fails.
	VerifyPassword(loginID, password string) (ClientProfile, error)
}
