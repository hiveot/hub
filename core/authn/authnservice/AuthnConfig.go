package authnservice

import (
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"path"
	"time"
)

// AuthnConfig contains the authn service configuration
type AuthnConfig struct {
	// PasswordFile with the file based password store.
	// Use a relative path for using the default $HOME/stores/authn location
	// Use "" for default defined in 'authnstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile,omitempty"`
	// Encryption of passwords: "argon2id" (default) or "bcrypt"
	Encryption string `yaml:"encryption,omitempty"`

	// Auth token validity for devices in seconds
	DeviceTokenValidity time.Duration `yaml:"deviceTokenValidity,omitempty"`
	// Auth token validity for services in seconds
	ServiceTokenValidity time.Duration `yaml:"serviceTokenValidity,omitempty"`
	// Auth token validity for users in seconds
	UserTokenValidity time.Duration `yaml:"userTokenValidity,omitempty"`

	// NoAutoStart prevents the authn service for auto starting. Intended for testing or custom implementation.
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`
}

// Setup ensures config is valid
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthnConfig) Setup(storesDir string) error {

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

	if cfg.DeviceTokenValidity == 0 {
		cfg.DeviceTokenValidity = auth.DefaultDeviceTokenValidity
	}
	if cfg.ServiceTokenValidity == 0 {
		cfg.ServiceTokenValidity = auth.DefaultServiceTokenValidity
	}
	if cfg.UserTokenValidity == 0 {
		cfg.UserTokenValidity = auth.DefaultUserTokenValidity
	}

	return nil
}
