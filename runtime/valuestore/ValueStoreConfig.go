package valuestore

const DefaultValueStoreFilename = "thingValues"

// ValueStoreConfig holds the configuration of the value store
type ValueStoreConfig struct {
	StoreFilename string `yaml:"storeFilename"`
}

// NewValueStoreConfig returns a new value store configuration with default values
func NewValueStoreConfig() ValueStoreConfig {
	cfg := ValueStoreConfig{
		StoreFilename: DefaultValueStoreFilename,
	}
	return cfg
}
