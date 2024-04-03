package directory

const DefaultDirStoreFilename = "directory.kvbtree"

// DirectoryConfig holds the configuration of the directory service
type DirectoryConfig struct {
	StoreFilename string `yaml:"storeFilename"`
}

// NewDirectoryConfig returns a directory configuration with default values
func NewDirectoryConfig() DirectoryConfig {
	cfg := DirectoryConfig{
		StoreFilename: DefaultDirStoreFilename,
	}
	return cfg
}
