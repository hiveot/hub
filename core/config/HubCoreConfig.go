package config

import (
	"github.com/nats-io/nkeys"
	"gopkg.in/yaml.v3"
	"os"
	"path"
)

// HubCoreConfig with Hub core configuration
// Use NewHubCoreConfig to create a default config
type HubCoreConfig struct {
	Server *ServerConfig `yaml:"server"`
	Authn  *AuthnConfig  `yaml:"authn"`
	//Authz  authz.AuthZConfig `yaml:"authz"`
}

// LoadConfig loads the hub core configuration from the given yaml file
// This only replaces the values that are defined in the config file
func (cfg *HubCoreConfig) LoadConfig(yamlFile string) error {
	data, err := os.ReadFile(yamlFile)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return err
	}
	// load the files defined in the server config
	err = cfg.Server.LoadConfig()
	if err != nil {
		return err
	}
	// load the files defined in the authn config
	err = cfg.Authn.LoadConfig()

	return err
}

// NewHubCoreConfig creates a configuration for the hub server and core services
//
//	accountName is the default application account name. Use "" for default
//	accountKey is the default application key for creating auth tokens. Use nil for default.
//	certsDir is the default location of CA and server certificates
//	storesDir is the default location of the storage root (services will each have a subdir)
func NewHubCoreConfig(accountName string, accountKey nkeys.KeyPair, certsDir string, storesDir string) *HubCoreConfig {
	binDir := path.Base(path.Base(os.Args[0]))
	if certsDir == "" {
		path.Join(binDir, "..", "certs")
	}
	if storesDir == "" {
		path.Join(binDir, "..", "stores")
	}
	hCfg := &HubCoreConfig{
		Server: NewServerConfig(accountName, accountKey, certsDir, storesDir),
		Authn:  NewAuthnConfig(storesDir),
	}
	return hCfg
}
