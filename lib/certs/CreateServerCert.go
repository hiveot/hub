package certs

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

// DefaultServerCertValidityDays with validity of generated service certificates
const DefaultServerCertValidityDays = 100

// CreateServerCert create a server certificate, signed by the given CA, for use in hiveot services.
//
// The provided x509 certificate can be converted to a PEM text with:
//
//	  certPEM = certs.X509CertToPEM(cert)
//
//	serviceID is the unique service ID used as the CN. for example hostname-serviceName
//	ou is the organizational unit of the certificate
//	pubkeyPEM is the server's public key
//	names are the SAN names to include with the certificate, typically the service IP address or host names
//	validityDays is the duration the cert is valid for. Use 0 for default.
//	caCert is the CA certificate used to sign the certificate
//	caKey is the CA private key used to sign certificate
func CreateServerCert(
	serverID string, ou string, serverPubKey *ecdsa.PublicKey, names []string, validityDays int,
	caCert *x509.Certificate, caKey *ecdsa.PrivateKey) (
	cert *x509.Certificate, err error) {

	if serverID == "" || serverPubKey == nil || names == nil {
		err := fmt.Errorf("missing argument serviceID, servicePubKey, or names")
		logrus.Error(err)
		return nil, err
	}
	if validityDays == 0 {
		validityDays = DefaultServerCertValidityDays
	}

	// firefox complains if serial is the same as that of the CA. So generate a unique one based on timestamp.
	serial := time.Now().Unix() - 3
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Country:            []string{"CA"},
			Province:           []string{"BC"},
			Locality:           []string{"local"},
			Organization:       []string{"hiveot"},
			OrganizationalUnit: []string{ou},
			CommonName:         serverID,
		},
		NotBefore: time.Now().Add(-time.Second),
		NotAfter:  time.Now().AddDate(0, 0, validityDays),
		//NotBefore: time.Now(),
		//NotAfter:  time.Now().AddDate(0, 0, config.DefaultServiceCertDurationDays),

		KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,
		// allow use as both server and client cert
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},

		// TBD: for NATS client cert the clientID should be added to the DNS name or Email address
		// TODO test this: Add the clientID to the SAN for mapping to a user in NATS
		// source: https://stackoverflow.com/questions/26441547/go-how-do-i-add-an-extension-subjectaltname-to-a-x509-certificate
		// and: https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/tls_mutual_auth
		// one of these solutions:
		//DNSNames: []string{clientID},
		//EmailAddresses: []string{serverID},

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
	//certKey := certs.CreateECDSAKeys()
	// and the certificate itself
	certDer, err := x509.CreateCertificate(
		rand.Reader, template, caCert, serverPubKey, caKey)
	if err == nil {
		cert, err = x509.ParseCertificate(certDer)
	}

	// TODO: send Thing event (services are things too)
	return cert, err
}

// CreateClientCert generates a x509 client certificate with keys, signed by the CA
// intended for testing, not for production
//
//		cn is the certificate common name, usually the client ID
//		ou the organization.
//		pubKey is the owner public key for this certificate
//		caCert and caKey is the signing CA
//	 validityDays
func CreateClientCert(cn string, ou string, pubKey *ecdsa.PublicKey,
	caCert *x509.Certificate, caKey *ecdsa.PrivateKey, validityDays int) (cert *x509.Certificate, derBytes []byte, err error) {
	validity := time.Hour * time.Duration(24*validityDays)

	extkeyUsage := x509.ExtKeyUsageClientAuth
	keyUsage := x509.KeyUsageDigitalSignature
	serial := time.Now().Unix() - 2

	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Country:            []string{"CA"},
			Organization:       []string{"hiveot"},
			Province:           []string{"BC"},
			Locality:           []string{"local"},
			CommonName:         cn,
			OrganizationalUnit: []string{ou},
			Names:              make([]pkix.AttributeTypeAndValue, 0),
		},
		NotBefore:   time.Now().Add(-10 * time.Second),
		NotAfter:    time.Now().Add(validity),
		KeyUsage:    keyUsage,
		ExtKeyUsage: []x509.ExtKeyUsage{extkeyUsage},

		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Not for production. Ignore all but the first error. Testing would fail if this fails.
	certDerBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, pubKey, caKey)
	certPEMBuffer := new(bytes.Buffer)
	_ = pem.Encode(certPEMBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: certDerBytes})
	cert, _ = x509.ParseCertificate(certDerBytes)
	if err != nil {
		panic("CreateClientCert. Failed creating cert: " + err.Error())
	}
	return cert, certDerBytes, err
}

// CreateClientTLSCert generates a TLS client certificate with keys, signed by the CA
// intended for testing, not for production
//
//	cn is the certificate common name, usually the client ID
//	ou the organization.
//	clientKey is the owner private key for this certificate
//	caCert and caKey is the signing CA
//	validityDays of the client certificate
func CreateClientTLSCert(cn string, ou string, clientKey *ecdsa.PrivateKey,
	caCert *x509.Certificate, caKey *ecdsa.PrivateKey, validityDays int) (
	cert *tls.Certificate, err error) {

	_, certDer, err := CreateClientCert(
		cn, ou, &clientKey.PublicKey, caCert, caKey, validityDays)

	// combined them into a TLS certificate
	tlsCert := &tls.Certificate{}
	tlsCert.Certificate = append(tlsCert.Certificate, certDer)
	tlsCert.PrivateKey = clientKey
	return tlsCert, err
}
