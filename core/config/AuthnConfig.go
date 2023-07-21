package config

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"path"
)

// AuthnConfig contains the svc service configuration
type AuthnConfig struct {
	// PasswordFile with the file based password store.
	// Use a relative path for using the default stores folder $HOME/stores/authn
	// Use "" for default defined in 'unpwstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile"`

	// Folder with CA certificate for clientcert based auth
	CertsDir string `yaml:"certsDir"`

	// Auth token validity for devices in seconds
	DeviceTokenValidity int `yaml:"deviceTokenValidity"`
	// Auth token validity for services in seconds
	ServiceTokenValidity int `yaml:"serviceTokenValidity"`
	// Auth token validity for users in seconds
	UserTokenValidity int `yaml:"userTokenValidity"`

	// NoAutoStart
	NoAutoStart bool `yaml:"noAutoStart"`
}

// InitConfig loads/creates missing files or folder if needed
func (cfg *AuthnConfig) InitConfig(certsDir string, storesDir string) error {

	if cfg.CertsDir == "" {
		cfg.CertsDir = certsDir
	}
	if cfg.PasswordFile == "" {
		cfg.PasswordFile = path.Join(storesDir, authn.AuthnServiceName, unpwstore.DefaultPasswordFile)
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
