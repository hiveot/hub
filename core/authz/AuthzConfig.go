package authz

import (
	"github.com/hiveot/hub/api/go/authz"
	"path"
)

// AuthzConfig contains the authorization service configuration
type AuthzConfig struct {
	// location of the group storage. Default is $HOME/stores/authz
	GroupsDir string `yaml:"groupsDir,omitempty"`

	// Do not auto-start the service. Intended for testing or custom implementation.
	NoAutoStart bool `yaml:"noAutoStart,omitempty"`
}

// InitConfig loads/creates missing files or folder if needed
func (cfg *AuthzConfig) InitConfig(storesDir string) error {
	// ensure all fields are properly set
	if cfg.GroupsDir == "" {
		cfg.GroupsDir = path.Join(storesDir, authz.AuthzServiceName)
	}
	return nil
}
