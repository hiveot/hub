package selfsigned

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/hiveot/hub/core/certs"
	certs2 "github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"log/slog"
	"math/big"
	"net"
	"time"
)

// SelfSignedCertsService creates certificates for use by services, devices and admin users.
//
// # This implements the ICertsService interface
//
// Note that this service does not support certificate revocation.
//
//	See also: https://www.imperialviolet.org/2014/04/19/revchecking.html
//
// Issued certificates are short-lived and must be renewed before they expire.
type SelfSignedCertsService struct {
	caCert     *x509.Certificate
	caCertPEM  string
	caKey      *ecdsa.PrivateKey
	caCertPool *x509.CertPool

	// messaging client for receiving requests
	hc hubclient.IHubClient
	// subscription to receive requests
	mngSub hubclient.ISubscription
}

// _createDeviceCert internal function to create a CA signed certificate for mutual authentication by IoT devices
func (svc *SelfSignedCertsService) _createDeviceCert(
	deviceID string, pubKey *ecdsa.PublicKey, validityDays int) (
	cert *x509.Certificate, err error) {
	if validityDays == 0 {
		validityDays = certs.DefaultDeviceCertValidityDays
	}

	cert, err = createClientCert(
		deviceID,
		certs2.OUIoTDevice,
		pubKey,
		svc.caCert,
		svc.caKey,
		validityDays)

	// TODO: send Thing event (services are things too)
	return cert, err
}

// createServiceCert internal function to create a CA signed service certificate for mutual authentication between services
func (svc *SelfSignedCertsService) _createServiceCert(
	serviceID string, servicePubKey *ecdsa.PublicKey, names []string, validityDays int) (
	cert *x509.Certificate, err error) {

	if serviceID == "" || servicePubKey == nil || names == nil {
		err := fmt.Errorf("missing argument serviceID, servicePubKey, or names")
		slog.Error(err.Error())
		return nil, err
	}
	if validityDays == 0 {
		validityDays = certs.DefaultServiceCertValidityDays
	}

	// firefox complains if serial is the same as that of the CA. So generate a unique one based on timestamp.
	serial := time.Now().Unix() - 3
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Country:            []string{"CA"},
			Province:           []string{"BC"},
			Locality:           []string{"local"},
			Organization:       []string{"HiveOT"},
			OrganizationalUnit: []string{certs2.OUService},
			CommonName:         serviceID,
		},
		NotBefore: time.Now().Add(-time.Second),
		NotAfter:  time.Now().AddDate(0, 0, validityDays),
		//NotBefore: time.Now(),
		//NotAfter:  time.Now().AddDate(0, 0, config.DefaultServiceCertDurationDays),

		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		//ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IsCA:           false,
		MaxPathLenZero: true,
		// BasicConstraintsValid: true,
		// IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		IPAddresses: []net.IP{},
	}
	// determine the hosts for this hub
	for _, h := range names {
		if ip := net.ParseIP(h); ip != nil {
			template.IPAddresses = append(template.IPAddresses, ip)
		} else {
			template.DNSNames = append(template.DNSNames, h)
		}
	}
	// Create the service private key
	//certKey := certs2.CreateECDSAKeys()
	// and the certificate itself
	certDer, err := x509.CreateCertificate(rand.Reader, template,
		svc.caCert, servicePubKey, svc.caKey)
	if err == nil {
		cert, err = x509.ParseCertificate(certDer)
	}

	// TODO: send Thing event (services are things too)
	return cert, err
}

// _createUserCert internal function to create a client certificate for end-users
func (svc *SelfSignedCertsService) _createUserCert(userID string, pubKey *ecdsa.PublicKey, validityDays int) (
	cert *x509.Certificate, err error) {
	if validityDays == 0 {
		validityDays = certs.DefaultUserCertValidityDays
	}

	cert, err = createClientCert(
		userID,
		certs2.OUUser,
		pubKey,
		svc.caCert,
		svc.caKey,
		validityDays)
	// TODO: send Thing event (services are things too)
	return cert, err
}

// CreateDeviceCert creates a CA signed certificate for mutual authentication by IoT devices in PEM format
func (svc *SelfSignedCertsService) CreateDeviceCert(
	args certs.CreateDeviceCertArgs) (certs.CreateCertResp, error) {
	//deviceID string, pubKeyPEM string, durationDays int) (
	//certPEM string, caCertPEM string, err error) {
	var cert *x509.Certificate

	slog.Info("CreateDeviceCert", "deviceID", args.DeviceID, "pubKey", args.PubKeyPEM)
	pubKey, err := certs2.PublicKeyFromPEM(args.PubKeyPEM)
	if err != nil {
		err = fmt.Errorf("public key for '%s' is invalid: %s", args.DeviceID, err)
	} else {
		cert, err = svc._createDeviceCert(args.DeviceID, pubKey, args.ValidityDays)
	}
	resp := certs.CreateCertResp{}
	if err == nil {
		resp.CertPEM = certs2.X509CertToPEM(cert)
		resp.CaCertPEM = svc.caCertPEM
	}
	return resp, err
}

// CreateServiceCert creates a CA signed service certificate for mutual authentication between services
func (svc *SelfSignedCertsService) CreateServiceCert(
	args certs.CreateServiceCertArgs) (certs.CreateCertResp, error) {
	var cert *x509.Certificate

	slog.Info("Creating service certificate",
		"serviceID", args.ServiceID, "names", args.Names)
	pubKey, err := certs2.PublicKeyFromPEM(args.PubKeyPEM)
	if err == nil {
		cert, err = svc._createServiceCert(
			args.ServiceID, pubKey, args.Names, args.ValidityDays,
		)
	}
	resp := certs.CreateCertResp{}
	if err == nil {
		resp.CertPEM = certs2.X509CertToPEM(cert)
		resp.CaCertPEM = svc.caCertPEM
	}
	// TODO: send event. is there a use-case for limiting service events to roles?
	return resp, err
}

// CreateUserCert creates a client certificate for end-users
func (svc *SelfSignedCertsService) CreateUserCert(
	args certs.CreateUserCertArgs) (certs.CreateCertResp, error) {
	//userID string, pubKeyPEM string, validityDays int) (
	//certPEM string, caCertPEM string, err error) {
	var cert *x509.Certificate

	slog.Info("CreateUserCert",
		"userID", args.UserID, "pubKey", args.PubKeyPEM)
	pubKey, err := certs2.PublicKeyFromPEM(args.PubKeyPEM)
	if err == nil {

		cert, err = svc._createUserCert(
			args.UserID, pubKey, args.ValidityDays)
	}
	resp := certs.CreateCertResp{}
	if err == nil {
		resp.CertPEM = certs2.X509CertToPEM(cert)
		resp.CaCertPEM = svc.caCertPEM
	}
	return resp, err
}

// Start the service and listen for requests
func (svc *SelfSignedCertsService) Start() (err error) {
	// for testing, hc can be nil
	if svc.hc != nil {
		svc.mngSub, err = hubclient.SubRPCCapability(certs.CertsManageCertsCapability,
			map[string]interface{}{
				certs.CreateDeviceCertReq:  svc.CreateDeviceCert,
				certs.CreateServiceCertReq: svc.CreateServiceCert,
				certs.CreateUserCertReq:    svc.CreateUserCert,
				certs.VerifyCertReq:        svc.VerifyCert,
			}, svc.hc)
		//svc.mngSub, err = svc.hc.SubRPCRequest(
		//	certs.CertsManageCertsCapability, svc.HandleRequest)
	}
	return err
}

// Stop the service and remove subscription
func (svc *SelfSignedCertsService) Stop() error {
	if svc.mngSub != nil {
		svc.mngSub.Unsubscribe()
		svc.mngSub = nil
	}
	return nil
}

// VerifyCert verifies whether the given certificate is a valid client certificate
func (svc *SelfSignedCertsService) VerifyCert(args certs.VerifyCertArgs) error {

	opts := x509.VerifyOptions{
		Roots:     svc.caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	cert, err := certs2.X509CertFromPEM(args.CertPEM)
	if err == nil {
		if cert.Subject.CommonName != args.ClientID {
			err = fmt.Errorf("client ID '%s' doesn't match certificate name '%s'",
				args.ClientID, cert.Subject.CommonName)
		}
	}
	//if err == nil {
	//	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	//}
	if err == nil {
		_, err = cert.Verify(opts)
	}
	return err
}

// NewSelfSignedCertsService returns a new instance of the selfsigned certificate service
//
//	caCert is the CA certificate used to created certificates
//	caKey is the CA private key used to created certificates
//	hc is the connection to the hub with a service role. For testing it can be nil.
func NewSelfSignedCertsService(
	caCert *x509.Certificate,
	caKey *ecdsa.PrivateKey,
	hc hubclient.IHubClient,
) *SelfSignedCertsService {

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(caCert)

	// Use one service instance per capability.
	// This does open the door to creating an instance per client session with embedded constraints,
	// although this is not needed at the moment.
	service := &SelfSignedCertsService{
		caCert:     caCert,
		caKey:      caKey,
		caCertPEM:  certs2.X509CertToPEM(caCert),
		caCertPool: caCertPool,
		hc:         hc,
	}
	if caCert == nil || caKey == nil || caCert.PublicKey == nil {
		panic("Missing CA certificate or key")
	}

	return service
}
