package unpwstore

// supported password hashes
const (
	PWHASH_ARGON2id = "argon2id"
	PWHASH_BCRYPT   = "bcrypt" // fallback in case argon2i cannot be used
)

// DefaultPasswordFile is the recommended password filename for Hub authentication
const DefaultPasswordFile = "hub.passwd"

// PasswordEntry containing hash and other user info
type PasswordEntry struct {
	// User login email
	LoginID string
	// password encrypted with argon2id or bcrypt
	PasswordHash string
	// user friendly name
	UserName string
	// timestamp password was last updated in seconds since epoch
	Updated int64
}

// IUnpwStore defined the interface for accessing the username-password store
type IUnpwStore interface {

	// Close the store
	Close()

	// Exists returns whether the given loginID already exists
	Exists(loginID string) bool

	// GetPasswordHash returns the password hash for the user, or "" if the user is not found
	//GetPasswordHash(username string) string

	// GetEntry returns the password entry of a user
	// Returns an error if the loginID doesn't exist
	GetEntry(loginID string) (entry PasswordEntry, err error)

	// List users in the store
	List() (entries []PasswordEntry, err error)

	// Open the store
	Open() error

	// Remove the user from the store
	// If the user doesn't exist, no error is returned
	Remove(userID string) (err error)

	// SetName stores the display name of a user
	// If the loginID doesn't exist, it will be added.
	SetName(loginID, name string) error

	// SetPassword stores the hash of the password for the given user.
	// If the loginID doesn't exist, it will be added.
	//
	// The hashing algorithm is embedded in the store.
	//  loginID is the login ID of the user whose hash to write
	//  password is the password whose hash to store
	// Returns error if the store isn't writable
	SetPassword(loginID string, password string) error

	// VerifyPassword verifies the given password against the stored hash
	// Returns the password entry for the login and an error if the verification fails.
	VerifyPassword(loginID, password string) (PasswordEntry, error)
}
