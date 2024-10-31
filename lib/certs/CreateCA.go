package certs

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"github.com/hiveot/hub/lib/keys"
	"log/slog"
	"math/big"
	"time"
)

const CertOrgName = "HiveOT"
const CertOrgLocality = "HiveOT zone"

// CreateCA creates a CA certificate with an private key for self-signed server certificates.
// Browsers don't support ed25519 keys so use ecdsa for the certificate.
// Source: https://shaneutt.com/blog/golang-ca-and-signed-cert-go/
func CreateCA(cn string, validityDays int) (cert *x509.Certificate, key keys.IHiveKey, err error) {

	// set up our CA certificate
	// see also: https://superuser.com/questions/738612/openssl-ca-keyusage-extension
	// firefox complains if serial is the same as that of the CA. So generate a unique one based on timestamp.
	serial := time.Now().Unix() - 1 // prevent duplicate timestamp with server cert
	rootTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(serial),
		Subject: pkix.Name{
			Country:      []string{"CA"},
			Province:     []string{"BC"},
			Locality:     []string{CertOrgLocality},
			Organization: []string{CertOrgName},
			CommonName:   cn,
		},
		NotBefore: time.Now().Add(-3 * time.Second),
		NotAfter:  time.Now().AddDate(0, 0, validityDays),
		// CA cert can be used to sign certificate and revocation lists
		KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature | x509.KeyUsageCRLSign | x509.KeyUsageDataEncipherment | x509.KeyUsageKeyEncipherment,

		// firefox seems to consider a CA invalid if extended key usage is combined with regular (critical) key usage???
		// certificate.Verify however fails if ext key usage is just the OCSPSigning.
		//ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageOCSPSigning},
		//ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageOCSPSigning},
		// https://github.com/hashicorp/vault/issues/846 suggests no ext key usage for CA's
		ExtKeyUsage: []x509.ExtKeyUsage{},

		// This hub cert is the only CA. Not using intermediate CAs
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Create the CA private key. Browsers don't support ed25519 (2024) so use ecdsa
	caKey := keys.NewKey(keys.KeyTypeECDSA)
	privKey := caKey.PrivateKey().(*ecdsa.PrivateKey)
	pubKey := caKey.PublicKey().(*ecdsa.PublicKey)

	// create the CA
	caCertDer, err := x509.CreateCertificate(
		rand.Reader, rootTemplate, rootTemplate, pubKey, privKey)
	if err != nil {
		// normally this never happens
		slog.Error("unable to create CA cert", "err", err)
	}
	caCert, _ := x509.ParseCertificate(caCertDer)
	return caCert, caKey, nil
}
