package middleware

type MiddlewareConfig struct {
}

// NewMiddlewareConfig creates a default configuration for the Digital Twin Runtime middleware.
func NewMiddlewareConfig() MiddlewareConfig {
	cfg := MiddlewareConfig{}
	return cfg
}
