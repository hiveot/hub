package directory

// DirectoryConfig holds the configuration of the directory service
type DirectoryConfig struct {
}

// NewDirectoryConfig returns a directory configuration with default values
func NewDirectoryConfig() DirectoryConfig {
	cfg := DirectoryConfig{}
	return cfg
}
