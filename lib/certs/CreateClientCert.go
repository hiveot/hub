package certs

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"time"

	"github.com/hiveot/hub/lib/keys"
)

// DefaultClientCertValidityDays with validity of generated service certificates
const DefaultClientCertValidityDays = 366

// CreateClientCert generates a x509 client certificate with keys, signed by the CA
// intended for testing, not for production
//
//	cn is the certificate common name, usually the client ID
//	ou the organization.
//	pubKey is the owner public key for this certificate
//	caKeys is the signing CA's key pair
//	validityDays
func CreateClientCert(cn string, ou string, validityDays int, pubKey keys.IHiveKey,
	caCert *x509.Certificate, caKeys keys.IHiveKey) (cert *x509.Certificate, err error) {
	validity := time.Hour * time.Duration(24*validityDays)

	if validityDays == 0 {
		validityDays = DefaultClientCertValidityDays
	}

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

	certDerBytes, err := x509.CreateCertificate(rand.Reader, template, caCert, pubKey.PublicKey(), caKeys.PrivateKey())
	if err == nil {
		//certPEMBuffer := new(bytes.Buffer)
		//_ = pem.Encode(certPEMBuffer, &pem.Block{Type: "CERTIFICATE", Bytes: certDerBytes})
		cert, err = x509.ParseCertificate(certDerBytes)
	}
	return cert, err
}
