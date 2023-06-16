package authn

// ServiceName of the service used for logging and connecting
const ServiceName = "authn"

// DefaultAccessTokenValiditySec with access token validity in seconds
const DefaultAccessTokenValiditySec = 300

// DefaultRefreshTokenValiditySec with Refresh token validity before refresh (14 days)
const DefaultRefreshTokenValiditySec = 30 * 24 * 3600

// UserProfile contains user information
type UserProfile struct {
	// The user's login ID, typically email
	LoginID string
	// The user's presentation name
	Name string
	// Last updated password in unix time
	Updated int64
}

// IAuthnService defines the interface for simple user management and authentication
type IAuthnService interface {

	// AddUser adds a user and generates a temporary password if one isn't given
	// If the loginID already exists then an error is returned
	// Users can set their own user name with IUserAuth
	AddUser(clientID string, password string) (newPassword string, err error)

	// GetProfile returns the user's profile
	// Login or Refresh must be called successfully first.
	GetProfile(clientID string) (profile UserProfile, err error)

	// ListUsers provide a list of users and their info
	ListUsers() (profiles []UserProfile, err error)

	// Login to authenticate a user
	// This returns a short lived auth token for use with the HTTP api,
	// and a medium lived refresh token used to obtain a new auth token.
	Login(clientID string, password string) (authToken, refreshToken string, err error)

	// Logout invalidates the refresh token
	Logout(clientID string, refreshToken string) (err error)

	// Refresh an authentication token
	// Refresh can be used instead of Login to authenticate and access the profile
	// refreshToken must be a valid refresh token obtained at login
	// This returns a short lived auth token and medium lived refresh token
	Refresh(clientID string, refreshToken string) (newAuthToken, newRefreshToken string, err error)

	// RemoveUser removes a user and disables login
	// Existing tokens are immediately expired (tbd)
	RemoveUser(clientID string) error

	// ResetPassword reset the user's password and returns a new password
	// the given password is optional. Use "" to generate a password
	ResetPassword(clientID string, password string) (newPassword string, err error)

	// SetPassword changes the client password
	// Login or Refresh must be called successfully first.
	SetPassword(clientID string, newPassword string) error

	// SetProfile updates the user profile
	// Login or Refresh must be called successfully first.
	SetProfile(profile UserProfile) error

	// TBD add OAuth2 login support

}
