package config

import (
	"github.com/hiveot/hub/lib/buckets"
)

// HistoryConfig with history store database configuration
type HistoryConfig struct {
	// Bucket store ID of the backend to store
	// kvbtree, pebble (default), bbolt. See IBucketStore for details.
	Backend string `yaml:"backend"`

	// Bucket store location where to store the history
	StoreDirectory string `yaml:"storeDirectory"`

	// Default retention from config by event name
	//Retention []history.EventRetention `yaml:"retention"`
}

// NewHistoryConfig creates a new config with default values
func NewHistoryConfig(storeDirectory string) HistoryConfig {
	cfg := HistoryConfig{
		Backend:        buckets.BackendPebble,
		StoreDirectory: storeDirectory,
	}
	return cfg
}
