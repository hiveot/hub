package certs_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/hiveot/hub/lib/certs"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadPrivKey(t *testing.T) {
	privKey, _ := certs.CreateECDSAKeys()
	err := certs.SaveKeysToPEM(privKey, testPrivKeyPemFile)
	assert.NoError(t, err)

	privKey2, err := certs.LoadKeysFromPEM(testPrivKeyPemFile)
	assert.NoError(t, err)
	assert.NotNil(t, privKey2)
}

func TestSaveLoadPubkey(t *testing.T) {
	privKey, _ := certs.CreateECDSAKeys()
	err := certs.SavePublicKeyToPEM(&privKey.PublicKey, testPubKeyPemFile)
	require.NoError(t, err)

	pubKey, err := certs.LoadPublicKeyFromPEM(testPubKeyPemFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, pubKey)
}

func TestSaveLoadPrivKeyNotFound(t *testing.T) {
	privKey, _ := certs.CreateECDSAKeys()
	// no access
	err := certs.SaveKeysToPEM(privKey, "/root")
	assert.Error(t, err)

	//
	privKey2, err := certs.LoadKeysFromPEM("/filedoesnotexist.pem")
	assert.Error(t, err)
	assert.Nil(t, privKey2)
}

func TestSaveLoadPubKeyNotFound(t *testing.T) {
	key, err := certs.LoadPublicKeyFromPEM("/filedoesnotexist.pem")
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestPublicKeyPEM(t *testing.T) {
	privKey, _ := certs.CreateECDSAKeys()

	pem, err := certs.PublicKeyToPEM(&privKey.PublicKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, pem)

	pubKey, err := certs.PublicKeyFromPEM(pem)
	assert.NoError(t, err)
	require.NotNil(t, pubKey)

	isEqual := privKey.PublicKey.Equal(pubKey)
	assert.True(t, isEqual)
}

func TestPrivateKeyPEM(t *testing.T) {
	privKey, _ := certs.CreateECDSAKeys()

	pem, err := certs.PrivateKeyToPEM(privKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, pem)

	privKey2, err := certs.PrivateKeyFromPEM(pem)
	assert.NoError(t, err)
	require.NotNil(t, privKey2)

	isEqual := privKey.Equal(privKey2)
	assert.True(t, isEqual)
}

func TestInvalidPEM(t *testing.T) {
	privKey, err := certs.PrivateKeyFromPEM("PRIVATE KEY")
	assert.Error(t, err)
	assert.Nil(t, privKey)

	pubKey, err := certs.PublicKeyFromPEM("PUBLIC KEY")
	assert.Error(t, err)
	assert.Nil(t, pubKey)

	//- part 2 switches public/private pem
	keys, _ := certs.CreateECDSAKeys()
	privPEM, err := certs.PrivateKeyToPEM(keys)
	assert.NoError(t, err)
	_, err = certs.PublicKeyFromPEM(privPEM)
	assert.Error(t, err)

	pubPEM, err := certs.PublicKeyToPEM(&keys.PublicKey)
	assert.NoError(t, err)
	_, err = certs.PrivateKeyFromPEM(pubPEM)
	assert.Error(t, err)
}

func TestWrongKeyFormat(t *testing.T) {
	keys, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	privPEM, err := certs.PrivateKeyToPEM(keys)
	assert.NoError(t, err)
	pubPEM, err := certs.PublicKeyToPEM(&keys.PublicKey)
	assert.NoError(t, err)

	// wrong key format should not panic
	_, err = certs.PrivateKeyFromPEM(privPEM)
	assert.Error(t, err)
	_, err = certs.PublicKeyFromPEM(pubPEM)
	assert.Error(t, err)

	_, err = certs.X509CertFromPEM("not a real pem")
	assert.Error(t, err)
}
