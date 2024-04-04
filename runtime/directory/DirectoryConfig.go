package directory

import "time"

const DefaultDirStoreFilename = "directory.kvbtree"

// DirectoryConfig holds the configuration of the directory service
type DirectoryConfig struct {
	// name of file or folder for the storage engine
	StoreFilename string `yaml:"storeFilename"`

	// CursorLifespan is the maximum lifespan of iteration cursors after last use in seconds.
	// This defaults to 10 seconds
	CursorLifespan time.Duration `yaml:"cursorLifespan,omitempty"`
}

// NewDirectoryConfig returns a directory configuration with default values
func NewDirectoryConfig() DirectoryConfig {
	cfg := DirectoryConfig{
		StoreFilename:  DefaultDirStoreFilename,
		CursorLifespan: 10, // seconds
	}
	return cfg
}
