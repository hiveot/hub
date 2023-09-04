package authservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"path"
)

// AuthConfig contains the auth service configuration
type AuthConfig struct {
	// PasswordFile with the file based password store.
	// Use a relative path for using the default $HOME/stores/authn location
	// Use "" for default defined in 'authnstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile,omitempty"`
	// Encryption of passwords: "argon2id" (default) or "bcrypt"
	Encryption string `yaml:"encryption,omitempty"`

	// Auth token validity for devices in days
	DeviceTokenValidityDays int `yaml:"deviceTokenValidityDays,omitempty"`
	// Auth token validity for services in days
	ServiceTokenValidityDays int `yaml:"serviceTokenValidityDays,omitempty"`
	// Auth token validity for users in days
	UserTokenValidityDays int `yaml:"userTokenValidityDays,omitempty"`

	// NoAutoStart prevents the auth service for auto starting. Intended for testing or custom implementation.
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`
}

// Setup ensures config is valid
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthConfig) Setup(storesDir string) error {

	if cfg.PasswordFile == "" {
		cfg.PasswordFile = auth.DefaultPasswordFile
	}
	if !path.IsAbs(cfg.PasswordFile) {
		cfg.PasswordFile = path.Join(storesDir, "authn", cfg.PasswordFile)
	}

	if cfg.Encryption == "" {
		cfg.Encryption = auth.PWHASH_ARGON2id
	}
	if cfg.Encryption != auth.PWHASH_BCRYPT && cfg.Encryption != auth.PWHASH_ARGON2id {
		return fmt.Errorf("unknown password encryption method: %s", cfg.Encryption)
	}

	if cfg.DeviceTokenValidityDays == 0 {
		cfg.DeviceTokenValidityDays = auth.DefaultDeviceTokenValidityDays
	}
	if cfg.ServiceTokenValidityDays == 0 {
		cfg.ServiceTokenValidityDays = auth.DefaultServiceTokenValidityDays
	}
	if cfg.UserTokenValidityDays == 0 {
		cfg.UserTokenValidityDays = auth.DefaultUserTokenValidityDays
	}

	return nil
}
