package auth

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

	// Setup for an administrator account
	AdminUserKeyFile   string `yaml:"adminUserKeyFile,omitempty"`   // default: admin.key
	AdminUserTokenFile string `yaml:"adminUserTokenFile,omitempty"` // default: admin.token

	// Setup for an launcher account
	LauncherKeyFile   string `yaml:"launcherKeyFile,omitempty"`   // default: launcher.key
	LauncherTokenFile string `yaml:"launcherTokenFile,omitempty"` // default: launcher.token
}

// Setup ensures config is valid
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthConfig) Setup(keysDir, storesDir string) error {

	if cfg.PasswordFile == "" {
		cfg.PasswordFile = auth.DefaultPasswordFile
	}
	if !path.IsAbs(cfg.PasswordFile) {
		cfg.PasswordFile = path.Join(storesDir, "auth", cfg.PasswordFile)
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

	if cfg.AdminUserKeyFile == "" {
		cfg.AdminUserKeyFile = auth.DefaultAdminUserID + ".key"
	}
	if !path.IsAbs(cfg.AdminUserKeyFile) {
		cfg.AdminUserKeyFile = path.Join(keysDir, cfg.AdminUserKeyFile)
	}

	if cfg.AdminUserTokenFile == "" {
		cfg.AdminUserTokenFile = auth.DefaultAdminUserID + ".token"
	}
	if !path.IsAbs(cfg.AdminUserTokenFile) {
		cfg.AdminUserTokenFile = path.Join(keysDir, cfg.AdminUserTokenFile)
	}

	if cfg.LauncherKeyFile == "" {
		cfg.LauncherKeyFile = auth.DefaultLauncherServiceID + ".key"
	}
	if !path.IsAbs(cfg.LauncherKeyFile) {
		cfg.LauncherKeyFile = path.Join(keysDir, cfg.LauncherKeyFile)
	}
	if cfg.LauncherTokenFile == "" {
		cfg.LauncherTokenFile = auth.DefaultLauncherServiceID + ".token"
	}
	if !path.IsAbs(cfg.LauncherTokenFile) {
		cfg.LauncherTokenFile = path.Join(keysDir, cfg.LauncherTokenFile)
	}
	return nil
}
