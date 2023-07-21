package config

import (
	"fmt"
	"github.com/hiveot/hub/lib/svcconfig"
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

// HubCoreConfig with Hub core configuration
// Use NewHubCoreConfig to create a default config
type HubCoreConfig struct {
	Server ServerConfig `yaml:"server"`
	Authn  AuthnConfig  `yaml:"authn"`
	Authz  AuthzConfig  `yaml:"authz"`
}

// NewHubCoreConfig creates a configuration for the hub server and core services
//
//	home dir of the application home. Default is the parent of the application bin folder
//	configfile with the name of the config file or "" to not load a config
func NewHubCoreConfig(home string, configFile string) (*HubCoreConfig, error) {
	f := svcconfig.GetFolders(home, false)
	if home == "" {
		home = path.Base(path.Base(os.Args[0]))
	}
	hubCfg := &HubCoreConfig{}
	// load config file
	if _, err := os.Stat(configFile); err == nil {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return hubCfg, fmt.Errorf("unable to load config: %w", err)
		}
		err = yaml.Unmarshal(data, hubCfg)
		if err != nil {
			return hubCfg, fmt.Errorf("unable to parse config: %w", err)
		}
	}
	// initialize the config by loading certificates and keys
	// load the files defined in the server config
	err := hubCfg.Server.InitConfig(f.Certs, f.Stores)
	if err != nil {
		return hubCfg, fmt.Errorf("server config error: %w", err)
	}
	// load the files defined in the authn config
	err = hubCfg.Authn.InitConfig(f.Certs, f.Stores)
	if err != nil {
		return hubCfg, fmt.Errorf("authn config error: %w", err)
	}

	// load the files defined in the authz config
	err = hubCfg.Authz.InitConfig(f.Stores)
	if err != nil {
		return hubCfg, fmt.Errorf("authz config error: %w", err)
	}

	return hubCfg, nil
}
