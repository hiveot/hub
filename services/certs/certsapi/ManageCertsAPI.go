// Package certs with POGS capability definitions of the certificate services.
// Unfortunately capnp does generate POGS types so we need to duplicate them
package certsapi

// DefaultServiceCertValidityDays with validity of generated service certificates
const DefaultServiceCertValidityDays = 100

// DefaultUserCertValidityDays with validity of generated client certificates
const DefaultUserCertValidityDays = 100

// DefaultDeviceCertValidityDays with validity of generated device certificates
const DefaultDeviceCertValidityDays = 100

// ServiceName to connect to the service
const ServiceName = "certs"

// ManageCertsCapability is the name of the Thing/Capability that handles management requests
const ManageCertsCapability = "manageCerts"

const CreateDeviceCertMethod = "createDeviceCert"

type CreateDeviceCertArgs struct {
	DeviceID     string `json:"deviceID"`
	PubKeyPEM    string `json:"pubKeyPEM"`
	ValidityDays int    `json:"validityDays"`
}
type CreateCertResp struct {
	CertPEM   string
	CaCertPEM string
}

const CreateServiceCertMethod = "createServiceCert"

type CreateServiceCertArgs struct {
	ServiceID    string   `json:"serviceID"`
	PubKeyPEM    string   `json:"pubKeyPEM"`
	Names        []string `json:"names"`
	ValidityDays int      `json:"validityDays"`
}

const CreateUserCertMethod = "createUserCert"

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
