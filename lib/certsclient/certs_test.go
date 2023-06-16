package certsclient_test

import (
	"path"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/testenv"
)

// ! These tests use the TestMain of main_test.go !

func TestX509ToFromPem(t *testing.T) {
	testCerts := testenv.CreateCertBundle()
	asPem := certsclient.X509CertToPEM(testCerts.CaCert)
	assert.NotEmpty(t, asPem)
	asX509, err := certsclient.X509CertFromPEM(asPem)
	assert.NoError(t, err)
	assert.NotEmpty(t, asX509)
}

func TestSaveLoadX509Cert(t *testing.T) {
	// hostnames := []string{"localhost"}
	caPemFile := path.Join(TestCertFolder, "caCert.pem")

	testCerts := testenv.CreateCertBundle()

	// save the test x509 cert
	err := certsclient.SaveX509CertToPEM(testCerts.CaCert, caPemFile)
	assert.NoError(t, err)

	cert, err := certsclient.LoadX509CertFromPEM(caPemFile)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestPublicKeyFromCert(t *testing.T) {
	testCerts := testenv.CreateCertBundle()
	pubKey := certsclient.PublicKeyFromCert(testCerts.CaCert)
	assert.NotEmpty(t, pubKey)
}

func TestSaveLoadTLSCert(t *testing.T) {
	// hostnames := []string{"localhost"}
	certFile := path.Join(TestCertFolder, "tlscert.pem")
	keyFile := path.Join(TestCertFolder, "tlskey.pem")

	testCerts := testenv.CreateCertBundle()

	// save the test x509 part of the TLS cert
	err := certsclient.SaveTLSCertToPEM(testCerts.DeviceCert, certFile, keyFile)
	assert.NoError(t, err)

	// load back the x509 part of the TLS cert
	cert, err := certsclient.LoadTLSCertFromPEM(certFile, keyFile)
	assert.NoError(t, err)
	assert.NotNil(t, cert)
}

func TestSaveLoadCertNoFile(t *testing.T) {
	certFile := "/root/notavalidcert.pem"
	keyFile := "/root/notavalidkey.pem"
	testCerts := testenv.CreateCertBundle()
	// save the test x509 cert
	err := certsclient.SaveX509CertToPEM(testCerts.CaCert, certFile)
	assert.Error(t, err)

	_, err = certsclient.LoadX509CertFromPEM(certFile)
	assert.Error(t, err)

	// save the test x509 part of the TLS cert
	err = certsclient.SaveTLSCertToPEM(testCerts.DeviceCert, certFile, keyFile)
	assert.Error(t, err)

	// load back the x509 part of the TLS cert
	_, err = certsclient.LoadTLSCertFromPEM(certFile, keyFile)
	assert.Error(t, err)

}
