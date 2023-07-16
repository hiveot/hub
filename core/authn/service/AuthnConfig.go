package service

import (
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"path"
)

// AuthnConfig contains the svc service configuration
type AuthnConfig struct {
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

	// Messaging server URL. Default is "tcp://localhost:4222"
	//ServerURL string `yaml:"address,omitempty"`

	// PasswordFile with the file based password store. Use "" for default defined in 'unpwstore.DefaultPasswordFile'
	PasswordFile string `yaml:"passwordFile,omitempty"`

	// Auth token validity for devices in seconds
	DeviceTokenValidity int `yaml:"deviceTokenValidity"`
	// Auth token validity for services in seconds
	ServiceTokenValidity int `yaml:"serviceTokenValidity"`
	// Auth token validity for users in seconds
	UserTokenValidity int `yaml:"userTokenValidity"`
}

// GetAuthKey returns the authentication key
// If needed load it from file
//func (cfg *AuthnConfig) GetAuthKey() (string, error) {
//	// load the authentication key to connect to the messaging server
//	if cfg.AuthKey == "" {
//		if cfg.AuthKeyFile == "" {
//			return "", errors.New("missing authentication key file")
//		}
//		authKeyBytes, err := os.ReadFile(cfg.AuthKeyFile)
//		if err != nil {
//			err2 := fmt.Errorf("error reading the authentication key file: %w", err)
//			return "", err2
//		}
//		cfg.AuthKey = string(authKeyBytes)
//	}
//	return cfg.AuthKey, nil
//}

// NewAuthnConfig returns a new instance of svc service configuration with defaults
//
//	storeFolder is the default directory for the stores
//
// func NewAuthnConfig(authKeyFile string, caCertFile string, storeFolder string) AuthnConfig {
func NewAuthnConfig(storeFolder string) AuthnConfig {
	cfg := AuthnConfig{
		//AuthKeyFile:          authKeyFile,
		//CaCertFile:           caCertFile,
		PasswordFile:         path.Join(storeFolder, authn.AuthnServiceName, unpwstore.DefaultPasswordFile),
		ServiceID:            authn.AuthnServiceName,
		DeviceTokenValidity:  authn.DefaultDeviceTokenValiditySec,
		ServiceTokenValidity: authn.DefaultServiceTokenValiditySec,
		UserTokenValidity:    authn.DefaultUserTokenValiditySec,
	}
	return cfg
}
