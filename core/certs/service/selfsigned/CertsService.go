package selfsigned

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/hiveot/hub/core/certs"
	certs2 "github.com/hiveot/hub/lib/certs"
	"math/big"
	"net"
	"time"

	"github.com/sirupsen/logrus"
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
}

// _createDeviceCert internal function to create a CA signed certificate for mutual authentication by IoT devices
func (srv *SelfSignedCertsService) _createDeviceCert(
	deviceID string, pubKey *ecdsa.PublicKey, validityDays int) (
	cert *x509.Certificate, err error) {
	if validityDays == 0 {
		validityDays = certs.DefaultDeviceCertValidityDays
	}

	cert, err = createClientCert(
		deviceID,
		certs2.OUIoTDevice,
		pubKey,
		srv.caCert,
		srv.caKey,
		validityDays)

	// TODO: send Thing event (services are things too)
	return cert, err
}

// createServiceCert internal function to create a CA signed service certificate for mutual authentication between services
func (srv *SelfSignedCertsService) _createServiceCert(
	serviceID string, servicePubKey *ecdsa.PublicKey, names []string, validityDays int) (
	cert *x509.Certificate, err error) {

	if serviceID == "" || servicePubKey == nil || names == nil {
		err := fmt.Errorf("missing argument serviceID, servicePubKey, or names")
		logrus.Error(err)
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
			Locality:           []string{CertOrgLocality},
			Organization:       []string{CertOrgName},
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
		srv.caCert, servicePubKey, srv.caKey)
	if err == nil {
		cert, err = x509.ParseCertificate(certDer)
	}

	// TODO: send Thing event (services are things too)
	return cert, err
}

// _createUserCert internal function to create a client certificate for end-users
func (srv *SelfSignedCertsService) _createUserCert(userID string, pubKey *ecdsa.PublicKey, validityDays int) (
	cert *x509.Certificate, err error) {
	if validityDays == 0 {
		validityDays = certs.DefaultUserCertValidityDays
	}

	cert, err = createClientCert(
		userID,
		certs2.OUUser,
		pubKey,
		srv.caCert,
		srv.caKey,
		validityDays)
	// TODO: send Thing event (services are things too)
	return cert, err
}

// CreateDeviceCert creates a CA signed certificate for mutual authentication by IoT devices in PEM format
func (srv *SelfSignedCertsService) CreateDeviceCert(
	deviceID string, pubKeyPEM string, durationDays int) (
	certPEM string, caCertPEM string, err error) {
	var cert *x509.Certificate

	logrus.Infof("deviceID='%s' pubKey='%s'", deviceID, pubKeyPEM)
	pubKey, err := certs2.PublicKeyFromPEM(pubKeyPEM)
	if err != nil {
		err = fmt.Errorf("public key for '%s' is invalid: %s", deviceID, err)
	} else {
		cert, err = srv._createDeviceCert(deviceID, pubKey, durationDays)
	}
	if err == nil {
		certPEM = certs2.X509CertToPEM(cert)
	}
	return certPEM, srv.caCertPEM, err
}

// CreateServiceCert creates a CA signed service certificate for mutual authentication between services
func (srv *SelfSignedCertsService) CreateServiceCert(
	serviceID string, pubKeyPEM string, names []string, validityDays int) (
	certPEM string, caCertPEM string, err error) {
	var cert *x509.Certificate

	logrus.Infof("Creating service certificate: serviceID='%s', names='%s'", serviceID, names)
	pubKey, err := certs2.PublicKeyFromPEM(pubKeyPEM)
	if err == nil {
		cert, err = srv._createServiceCert(
			serviceID,
			pubKey,
			names,
			validityDays,
		)
	}
	if err == nil {
		certPEM = certs2.X509CertToPEM(cert)
	}
	// TODO: send Thing event (services are things too)
	return certPEM, srv.caCertPEM, err
}

// CreateUserCert creates a client certificate for end-users
func (srv *SelfSignedCertsService) CreateUserCert(
	userID string, pubKeyPEM string, validityDays int) (
	certPEM string, caCertPEM string, err error) {
	var cert *x509.Certificate

	logrus.Infof("userID='%s' pubKey='%s'", userID, pubKeyPEM)
	pubKey, err := certs2.PublicKeyFromPEM(pubKeyPEM)
	if err == nil {

		cert, err = srv._createUserCert(
			userID,
			pubKey,
			validityDays)
	}
	if err == nil {
		certPEM = certs2.X509CertToPEM(cert)
	}

	// TODO: send Thing event (services are things too)
	return certPEM, srv.caCertPEM, err
}

// Start the service
func (srv *SelfSignedCertsService) Start() error {
	// nothing to do here
	return nil
}

// Stop the service
func (srv *SelfSignedCertsService) Stop() error {
	// nothing to do here
	return nil
}

// VerifyCert verifies whether the given certificate is a valid client certificate
func (srv *SelfSignedCertsService) VerifyCert(
	clientID string, certPEM string) error {

	opts := x509.VerifyOptions{
		Roots:     srv.caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	cert, err := certs2.X509CertFromPEM(certPEM)
	if err == nil {
		if cert.Subject.CommonName != clientID {
			err = fmt.Errorf("client ID '%s' doesn't match certificate name '%s'", clientID, cert.Subject.CommonName)
		}
	}
	//if err == nil {
	//	x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
	//}
	if err == nil {
		// FIXME: TestCertAuth: certificate specifies incompatible key usage
		// why? Is the certpool invalid? Yet the test succeeds
		_, err = cert.Verify(opts)
	}
	return err
}

// NewSelfSignedCertsService returns a new instance of the selfsigned certificate service
//
//	caCert is the CA certificate used to created certificates
//	caKey is the CA private key used to created certificates
func NewSelfSignedCertsService(caCert *x509.Certificate, caKey *ecdsa.PrivateKey) *SelfSignedCertsService {
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
	}
	if caCert == nil || caKey == nil || caCert.PublicKey == nil {
		logrus.Panic("Missing CA certificate or key")
	}

	return service
}
