package certsapi

// AgentID is the connect ID of the agent connecting to the Hub
const AgentID = "certs"

// ManageCertsServiceID is the ID of the service exposed by the agent
const ManageCertsServiceID = "manage"

// Management methods
const (
	CreateDeviceCertMethod  = "createDeviceCert"
	CreateServiceCertMethod = "createServiceCert"
	CreateUserCertMethod    = "createUserCert"
)

// DefaultServiceCertValidityDays with validity of generated service certificates
const DefaultServiceCertValidityDays = 100

// DefaultUserCertValidityDays with validity of generated client certificates
const DefaultUserCertValidityDays = 100

// DefaultDeviceCertValidityDays with validity of generated device certificates
const DefaultDeviceCertValidityDays = 100

type CreateDeviceCertArgs struct {
	DeviceID     string `json:"deviceID"`
	PubKeyPEM    string `json:"pubKeyPEM"`
	ValidityDays int    `json:"validityDays"`
}
type CreateCertResp struct {
	CertPEM   string
	CaCertPEM string
}

type CreateServiceCertArgs struct {
	ServiceID    string   `json:"serviceID"`
	PubKeyPEM    string   `json:"pubKeyPEM"`
	Names        []string `json:"names"`
	ValidityDays int      `json:"validityDays"`
}

type CreateUserCertArgs struct {
	UserID       string `json:"userID"`
	PubKeyPEM    string `json:"pubKeyPEM"`
	ValidityDays int    `json:"validityDays"`
}

const VerifyCertMethod = "verifyCert"

type VerifyCertArgs struct {
	ClientID string `json:"clientID"`
	CertPEM  string `json:"certPEM"`
}
