package certs

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"golang.org/x/exp/slog"
	"math/big"
	"net"
	"time"
)

// DefaultServerCertValidityDays with validity of generated service certificates
const DefaultServerCertValidityDays = 100

// CreateServerCert create a server certificate, signed by the given CA, for use in hiveot services.
//
// The provided x509 certificate can be converted to a PEM text with:
//
//	  certPEM = certs.X509CertToPEM(cert)
//
//	* serviceID is the unique service ID used as the CN. for example hostname-serviceName
//	* ou is the organizational unit of the certificate
//	* validityDays is the duration the cert is valid for. Use 0 for default.
//	* serverPubKey is the server's public key
//	* names are the SAN names to include with the certificate, localhost and 127.0.0.1 are always added
//	* caCert is the CA certificate used to sign the certificate
//	* caKey is the CA private key used to sign certificate
func CreateServerCert(
	serverID string, ou string, validityDays int,
	serverPubKey *ecdsa.PublicKey, names []string,
	caCert *x509.Certificate, caKey *ecdsa.PrivateKey) (
	x509Cert *x509.Certificate, err error) {

	if serverID == "" || serverPubKey == nil {
		err := fmt.Errorf("missing argument serviceID, servicePubKey")
		slog.Error(err.Error())
		return nil, err
	}
	if validityDays == 0 {
		validityDays = DefaultServerCertValidityDays
	}
	if names == nil {
		names = []string{}
	}
	names = append(names, "127.0.0.1")
	names = append(names, "localhost")

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
	certDerBytes, err := x509.CreateCertificate(
		rand.Reader, template, caCert, serverPubKey, caKey)
	if err == nil {
		x509Cert, err = x509.ParseCertificate(certDerBytes)
	}
	return x509Cert, err
}
