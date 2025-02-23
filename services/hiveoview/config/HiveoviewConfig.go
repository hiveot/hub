package config

// HiveoviewConfig with UI presets
type HiveoviewConfig struct {
	ServerPort int `yaml:"serverPort"`
}

// NewHiveoviewConfig
func NewHiveoviewConfig(serverPort int) HiveoviewConfig {
	cfg := HiveoviewConfig{
		ServerPort: serverPort,
	}
	return cfg
}
