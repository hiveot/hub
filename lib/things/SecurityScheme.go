package things

// SecurityScheme
type SecurityScheme struct {
	// JSON-LD keyword to label the object with semantic tags (or types).
	AtType []string `json:"@type,omitempty"`

	// Provides additional (human-readable) information based on a default language
	Description string `json:"description,omitempty"`

	// Can be used to support (human-readable) information in different languages. Also see MultiLanguage.
	Descriptions []string `json:"descriptions,omitempty"`

	// URI of the proxy server this security configuration provides access to.
	// If not given, the corresponding security configuration is for the endpoint.
	Proxy string `json:"proxy,omitempty"`

	// Identification of the security mechanism being configured.
	Scheme string `json:"scheme"`
}
