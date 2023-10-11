package config

import (
	"github.com/hiveot/hub/pkg/bucketstore"
	"github.com/hiveot/hub/pkg/history"
)

// HistoryConfig with history store database configuration
type HistoryConfig struct {
	// Bucket store ID of the backend to store
	// kvbtree, pebble (default), bbolt. See IBucketStore for details.
	Backend string `yaml:"backend"`

	// Bucket store location where to store the history
	Directory string `yaml:"directory"`

	// instance ID of the service, eg: "history".
	ServiceID string `yaml:"serviceID"`

	// Default retention from config by event name
	Retention []history.EventRetention `yaml:"retention"`
}

// NewHistoryConfig creates a new config with default values
func NewHistoryConfig(storeDirectory string) HistoryConfig {
	cfg := HistoryConfig{
		Backend:   bucketstore.BackendPebble,
		Directory: storeDirectory,
		ServiceID: history.ServiceName,
	}
	return cfg
}
