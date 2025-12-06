package selfsigned

import (
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"time"

	"github.com/hiveot/hivekit/go/lib/keys"
)

// createClientCert is the internal function to create a client certificate
// for IoT devices, administrator
//
// The client ouRole is intended to for role based authorization. It is stored in the
// certificate OrganizationalUnit. See OUxxx
//
// This generates a TLS client certificate with keys
//
//	clientID used as the CommonName, eg pluginID or deviceID
//	ouRole with type of client: OUNone, OUAdmin, OUClient, OUIoTDevice
//	ownerPubKey the public key of the certificate holder
//	caCert CA's certificate for signing
//	caPrivKey CA's ECDSA key for signing
//	validityDays nr of days the certificate will be valid
//
// Returns the signed certificate with the corresponding CA used to sign, or an error
func createClientCert(
	clientID string, ouRole string, ownerPubKey keys.IHiveKey,
	caCert *x509.Certificate, caPrivKey keys.IHiveKey, validityDays int) (
	clientCert *x509.Certificate, err error) {

	var newCert *x509.Certificate

	if clientID == "" || ownerPubKey == nil {
		err := fmt.Errorf("missing clientID or client public key")
		slog.Error(err.Error())
		return nil, err
	}
	// firefox complains if serial is the same as that of the CA. So generate a unique one based on timestamp.
	serial := time.Now().Unix() - 2
	template := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Country:            []string{"CA"},
			Province:           []string{"BC"},
			Locality:           []string{"local"},
			Organization:       []string{"HiveOT"},
			OrganizationalUnit: []string{ouRole},
			CommonName:         clientID,
			Names:              make([]pkix.AttributeTypeAndValue, 0),
		},
		NotBefore: time.Now().Add(-time.Second),
		NotAfter:  time.Now().AddDate(0, 0, validityDays),

		//KeyUsage: x509.KeyUsageDigitalSignature | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},

		BasicConstraintsValid: true,
		IsCA:                  false,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	// clientKey := certs.CreateECDSAKeys()
	certDer, err := x509.CreateCertificate(rand.Reader, template, caCert, ownerPubKey.PublicKey(), caPrivKey.PrivateKey())
	if err == nil {
		newCert, err = x509.ParseCertificate(certDer)
	}

	// // combined them into a TLS certificate
	// tlscert := &tls.Certificate{}
	// tlscert.Certificate = append(tlscert.Certificate, certDer)
	// tlscert.PrivateKey = clientKey

	return newCert, err
}
