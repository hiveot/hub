// Package certs with POGS capability definitions of the certificate services.
// Unfortunately capnp does generate POGS types so we need to duplicate them
package certs

// DefaultServiceCertValidityDays with validity of generated service certificates
const DefaultServiceCertValidityDays = 100

// DefaultUserCertValidityDays with validity of generated client certificates
const DefaultUserCertValidityDays = 100

// DefaultDeviceCertValidityDays with validity of generated device certificates
const DefaultDeviceCertValidityDays = 100

// ServiceName to connect to the service
const ServiceName = "certs"

// CertsManageCertsCapability is the name of the Thing/Capability that handles management requests
const CertsManageCertsCapability = "manage"

const CreateDeviceCertAction = "createDeviceCert"

type CreateDeviceCertReq struct {
	DeviceID     string `json:"deviceID"`
	PubKeyPEM    string `json:"pubKeyPEM"`
	ValidityDays int    `json:"validityDays"`
}
type CreateCertResp struct {
	CertPEM   string
	CaCertPEM string
}

const CreateServiceCertAction = "createServiceCert"

type CreateServiceCertReq struct {
	ServiceID    string   `json:"serviceID"`
	PubKeyPEM    string   `json:"pubKeyPEM"`
	Names        []string `json:"names"`
	ValidityDays int      `json:"validityDays"`
}

const CreateUserCertAction = "createUserCert"

type CreateUserCertReq struct {
	UserID       string `json:"userID"`
	PubKeyPEM    string `json:"pubKeyPEM"`
	ValidityDays int    `json:"validityDays"`
}

const VerifyCertAction = "verifyCert"

type VerifyCertReq struct {
	ClientID string `json:"clientID"`
	CertPEM  string `json:"certPEM"`
}

// ICertService defines the capability of managing certs
// This interface aggregates all certificate capabilities.
// This approach is experimental and intended to improve security by providing capabilities based on
// user credentials, enforced by the capnp protocol.
type ICertService interface {

	// CreateDeviceCert generates or renews IoT device certificate for access hub IoT gateway
	//  deviceID is the unique device's ID
	//  pubkeyPEM is the device's public key in PEM format
	//  validityDays is the duration the cert is valid for. Use 0 for default.
	CreateDeviceCert(deviceID string, pubKeyPEM string, validityDays int) (
		certPEM string, caCertPEM string, err error)

	// CreateServiceCert generates a hub service certificate
	// This returns the PEM encoded certificate with certificate of the CA that signed it.
	// An error is returned if one of the parameters is invalid.
	//
	//  serviceID is the unique service ID used as the CN. for example hostname-serviceName
	//  pubkeyPEM is the device's public key in PEM format
	//  names are the SAN names to include with the certificate, typically the service IP address or host names
	//  validityDays is the duration the cert is valid for. Use 0 for default.
	CreateServiceCert(serviceID string, pubKeyPEM string, names []string, validityDays int) (
		certPEM string, caCertPEM string, err error)

	// CreateUserCert generates an end-user certificate for access hub gateway services
	// Intended for users that use certificates instead of regular login.
	//  userID is the unique user's ID, for example an email address
	//  pubkeyPEM is the user's public key in PEM format
	//  validityDays is the duration the cert is valid for. Use 0 for default.
	CreateUserCert(userID string, pubKeyPEM string, validityDays int) (
		certPEM string, caCertPEM string, err error)

	// VerifyCert verifies if the certificate is valid for the Hub
	VerifyCert(clientID string, certPEM string) error
}
