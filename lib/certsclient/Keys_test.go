package certsclient_test

import (
	"crypto/rand"
	"crypto/rsa"
	"testing"

	"github.com/hiveot/hub/lib/certsclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveLoadPrivKey(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()
	err := certsclient.SaveKeysToPEM(privKey, testPrivKeyPemFile)
	assert.NoError(t, err)

	privKey2, err := certsclient.LoadKeysFromPEM(testPrivKeyPemFile)
	assert.NoError(t, err)
	assert.NotNil(t, privKey2)
}

func TestSaveLoadPubkey(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()
	err := certsclient.SavePublicKeyToPEM(&privKey.PublicKey, testPubKeyPemFile)
	require.NoError(t, err)

	pubKey, err := certsclient.LoadPublicKeyFromPEM(testPubKeyPemFile)
	assert.NoError(t, err)
	assert.NotEmpty(t, pubKey)
}

func TestSaveLoadPrivKeyNotFound(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()
	// no access
	err := certsclient.SaveKeysToPEM(privKey, "/root")
	assert.Error(t, err)

	//
	privKey2, err := certsclient.LoadKeysFromPEM("/filedoesnotexist.pem")
	assert.Error(t, err)
	assert.Nil(t, privKey2)
}

func TestSaveLoadPubKeyNotFound(t *testing.T) {
	key, err := certsclient.LoadPublicKeyFromPEM("/filedoesnotexist.pem")
	assert.Error(t, err)
	assert.Nil(t, key)
}

func TestPublicKeyPEM(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()

	pem, err := certsclient.PublicKeyToPEM(&privKey.PublicKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, pem)

	pubKey, err := certsclient.PublicKeyFromPEM(pem)
	assert.NoError(t, err)
	require.NotNil(t, pubKey)

	isEqual := privKey.PublicKey.Equal(pubKey)
	assert.True(t, isEqual)
}

func TestPrivateKeyPEM(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()

	pem, err := certsclient.PrivateKeyToPEM(privKey)

	assert.NoError(t, err)
	assert.NotEmpty(t, pem)

	privKey2, err := certsclient.PrivateKeyFromPEM(pem)
	assert.NoError(t, err)
	require.NotNil(t, privKey2)

	isEqual := privKey.Equal(privKey2)
	assert.True(t, isEqual)
}

func TestInvalidPEM(t *testing.T) {
	privKey, err := certsclient.PrivateKeyFromPEM("PRIVATE KEY")
	assert.Error(t, err)
	assert.Nil(t, privKey)

	pubKey, err := certsclient.PublicKeyFromPEM("PUBLIC KEY")
	assert.Error(t, err)
	assert.Nil(t, pubKey)

	//- part 2 switches public/private pem
	keys := certsclient.CreateECDSAKeys()
	privPEM, err := certsclient.PrivateKeyToPEM(keys)
	assert.NoError(t, err)
	_, err = certsclient.PublicKeyFromPEM(privPEM)
	assert.Error(t, err)

	pubPEM, err := certsclient.PublicKeyToPEM(&keys.PublicKey)
	assert.NoError(t, err)
	_, err = certsclient.PrivateKeyFromPEM(pubPEM)
	assert.Error(t, err)
}

func TestWrongKeyFormat(t *testing.T) {
	keys, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)
	privPEM, err := certsclient.PrivateKeyToPEM(keys)
	assert.NoError(t, err)
	pubPEM, err := certsclient.PublicKeyToPEM(&keys.PublicKey)
	assert.NoError(t, err)

	// wrong key format should not panic
	_, err = certsclient.PrivateKeyFromPEM(privPEM)
	assert.Error(t, err)
	_, err = certsclient.PublicKeyFromPEM(pubPEM)
	assert.Error(t, err)

	_, err = certsclient.X509CertFromPEM("not a real pem")
	assert.Error(t, err)
}
