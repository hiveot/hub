package config

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"os"
	"path"
)

// AuthnConfig contains the svc service configuration
type AuthnConfig struct {
	// The default account to create tokens for
	//AccountName string `yaml:"accountName"`

	// the account key file used to sign generated tokens
	//AccountKeyFile string `yaml:"accountKeyFile"`

	//// AuthKey is the service authentication token for connecting to the server
	//// This takes precedence over AuthKeyFile. If omitted then AuthKeyFile is used.
	//AuthKey string `yaml:"authKey,omitempty"`
	//
	//// AuthKeyFile is the file containing the service authentication JWT token for connecting to the server
	//// This is required when AuthKey is not provided.
	//AuthKeyFile string `yaml:"authKeyFile,omitempty"`

	// ID of the service for use in password file store and subject: (things.{serviceID}.*.action.{name}
	// The default ID for single instances is svc-{hostname}
	ServiceID string `yaml:"serviceID,omitempty"`

	// PasswordFile with the file based password store.
	// Use a relative path for using the default stores folder $HOME/stores/authn
	// Use "" for default defined in 'unpwstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile,omitempty"`

	// Auth token validity for devices in seconds
	DeviceTokenValidity int `yaml:"deviceTokenValidity"`
	// Auth token validity for services in seconds
	ServiceTokenValidity int `yaml:"serviceTokenValidity"`
	// Auth token validity for users in seconds
	UserTokenValidity int `yaml:"userTokenValidity"`
}

// LoadConfig loads the files used in the configuration
func (cfg *AuthnConfig) LoadConfig() (err error) {
	return err
}

// NewAuthnConfig returns a new instance of svc service configuration with defaults
//
//	accountName is the default application account name
//	certsDir is the default location of CA and server certificates
//	storesDir is the default location of the storage root (services will each have a subdir)
//
// func NewAuthnConfig(authKeyFile string, caCertFile string, storeFolder string) AuthnConfig {
func NewAuthnConfig(storesDir string) *AuthnConfig {
	hostName, _ := os.Hostname()
	cfg := &AuthnConfig{
		PasswordFile:         path.Join(storesDir, authn.AuthnServiceName, unpwstore.DefaultPasswordFile),
		ServiceID:            authn.AuthnServiceName + "-" + hostName,
		DeviceTokenValidity:  authn.DefaultDeviceTokenValiditySec,
		ServiceTokenValidity: authn.DefaultServiceTokenValiditySec,
		UserTokenValidity:    authn.DefaultUserTokenValiditySec,
	}
	return cfg
}
