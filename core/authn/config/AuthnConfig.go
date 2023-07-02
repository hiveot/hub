package config

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"path"
)

// AuthnConfig contains the authn service configuration
type AuthnConfig struct {

	// PasswordFile to read from. Use "" for default defined in 'unpwstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile"`

	// Access token validity. Default is 1 hour
	AccessTokenValiditySec int `yaml:"accessTokenValiditySec"`

	// Refresh token validity. Default is 1209600 (14 days)
	RefreshTokenValiditySec int `yaml:"refreshTokenValiditySec"`
}

// NewAuthnConfig returns a new instance of authn service configuration with defaults
//
//	storeFolder is the default directory for the stores
func NewAuthnConfig(storeFolder string) AuthnConfig {
	cfg := AuthnConfig{
		PasswordFile:            path.Join(storeFolder, authn.ServiceName, unpwstore.DefaultPasswordFile),
		AccessTokenValiditySec:  3600,
		RefreshTokenValiditySec: 1209600,
	}
	return cfg
}
