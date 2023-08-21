package authn

import (
	"github.com/hiveot/hub/api/go/authn"
	"path"
)

// AuthnConfig contains the authn service configuration
type AuthnConfig struct {
	// PasswordFile with the file based password store.
	// Use a relative path for using the default $HOME/stores/authn location
	// Use "" for default defined in 'authnstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile"`

	// Auth token validity for devices in seconds
	DeviceTokenValidity int `yaml:"deviceTokenValidity"`
	// Auth token validity for services in seconds
	ServiceTokenValidity int `yaml:"serviceTokenValidity"`
	// Auth token validity for users in seconds
	UserTokenValidity int `yaml:"userTokenValidity"`

	// NoAutoStart prevents the authn service for auto starting. Intended for testing or custom implementation.
	NoAutoStart bool `yaml:"noAutoStart"`
}

// Setup ensures config is valid
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthnConfig) Setup(storesDir string) error {

	if cfg.PasswordFile == "" {
		cfg.PasswordFile = authn.DefaultPasswordFile
	}
	if !path.IsAbs(cfg.PasswordFile) {
		cfg.PasswordFile = path.Join(storesDir, "authn", cfg.PasswordFile)
	}
	if cfg.DeviceTokenValidity == 0 {
		cfg.DeviceTokenValidity = authn.DefaultDeviceTokenValiditySec
	}
	if cfg.ServiceTokenValidity == 0 {
		cfg.ServiceTokenValidity = authn.DefaultServiceTokenValiditySec
	}
	if cfg.UserTokenValidity == 0 {
		cfg.UserTokenValidity = authn.DefaultUserTokenValiditySec
	}

	return nil
}
