package router

// RouterConfig for configuration of the router behavior
type RouterConfig struct {
	// this is a placeholder. Might actually use it in the future

	// TODO: list of active middleware such as logging and rate control
}

// NewRouterConfig creates a default configuration
func NewRouterConfig() RouterConfig {
	cfg := RouterConfig{}
	return cfg
}
