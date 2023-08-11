package certs_test

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hiveot/hub/lib/certs"
)

// ! These tests use the TestMain of main_test.go !

func TestX509ToFromPem(t *testing.T) {
	testCerts := certs.CreateCertBundle()
	asPem := certs.X509CertToPEM(testCerts.CaCert)
	assert.NotEmpty(t, asPem)
	asX509, err := certs.X509CertFromPEM(asPem)
	assert.NoError(t, err)
	assert.NotEmpty(t, asX509)
}

func TestSaveLoadX509Cert(t *testing.T) {
	// hostnames := []string{"localhost"}
	caPemFile := path.Join(TestCertFolder, "caCert.pem")

	testCerts := certs.CreateCertBundle()

	// save the test x509 cert
	err := certs.SaveX509CertToPEM(testCerts.CaCert, caPemFile)
	assert.NoError(t, err)

	cert, err := certs.LoadX509CertFromPEM(caPemFile)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestPublicKeyFromCert(t *testing.T) {
	testCerts := certs.CreateCertBundle()
	pubKey := certs.PublicKeyFromCert(testCerts.CaCert)
	assert.NotEmpty(t, pubKey)
}

func TestSaveLoadTLSCert(t *testing.T) {
	// hostnames := []string{"localhost"}
	certFile := path.Join(TestCertFolder, "tlscert.pem")
	keyFile := path.Join(TestCertFolder, "tlskey.pem")

	testCerts := certs.CreateCertBundle()

	// save the test x509 part of the TLS cert
	err := certs.SaveTLSCertToPEM(testCerts.ServerCert, certFile, keyFile)
	assert.NoError(t, err)

	// load back the x509 part of the TLS cert
	cert, err := certs.LoadTLSCertFromPEM(certFile, keyFile)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestSaveLoadCertNoFile(t *testing.T) {
	certFile := "/root/notavalidcert.pem"
	keyFile := "/root/notavalidkey.pem"
	testCerts := certs.CreateCertBundle()
	// save the test x509 cert
	err := certs.SaveX509CertToPEM(testCerts.CaCert, certFile)
	assert.Error(t, err)

	_, err = certs.LoadX509CertFromPEM(certFile)
	assert.Error(t, err)

	// save the test x509 part of the TLS cert
	err = certs.SaveTLSCertToPEM(testCerts.ServerCert, certFile, keyFile)
	assert.Error(t, err)

	// load back the x509 part of the TLS cert
	_, err = certs.LoadTLSCertFromPEM(certFile, keyFile)
	assert.Error(t, err)

}
