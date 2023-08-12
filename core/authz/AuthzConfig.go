package authz

import "path"

// AuthzConfig contains the authorization service configuration
type AuthzConfig struct {
	// location of the authz data storage. Default is $HOME/stores/authz
	DataDir string `yaml:"dataDir,omitempty"`

	// Do not auto-start the service. Intended for testing or custom implementation.
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`
}

// Setup loads/creates missing files or folder if needed
//
//	storesDir is the default storage root directory ($HOME/stores)
func (cfg *AuthzConfig) Setup(storesDir string) error {
	// ensure all fields are properly set
	if cfg.DataDir == "" {
		cfg.DataDir = path.Join(storesDir, "authz")
	}
	return nil
}
