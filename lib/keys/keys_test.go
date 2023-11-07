package keys_test

import (
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/logging"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const keyType = keys.KeyTypeECDSA

// set in TestMain
var TestKeysFolder string
var testPrivKeyPemFile string
var testPubKeyPemFile string

// TestMain create a test folder for keys
func TestMain(m *testing.M) {
	TestKeysFolder, _ = os.MkdirTemp("", "hiveot-keys-")

	testPrivKeyPemFile = path.Join(TestKeysFolder, "privKey.pem")
	testPubKeyPemFile = path.Join(TestKeysFolder, "pubKey.pem")
	logging.SetLogging("info", "")

	result := m.Run()
	if result != 0 {
		println("Test failed with code:", result)
		println("Find test files in:", TestKeysFolder)
	} else {
		// comment out the next line to be able to inspect results
		_ = os.RemoveAll(TestKeysFolder)
	}

	os.Exit(result)
}

func TestSaveLoadPrivKey(t *testing.T) {
	k1 := keys.NewKey(keyType)
	err := k1.ExportPrivateToFile(testPrivKeyPemFile)
	assert.NoError(t, err)

	k2, err := keys.NewKeyFromFile(testPrivKeyPemFile)
	assert.NoError(t, err)
	require.NotNil(t, k2)
	require.Equal(t, keyType, k2.KeyType())

	pem1 := k1.ExportPublic()
	pem2 := k2.ExportPublic()

	msg := []byte("hello world")
	signature, err := k1.Sign(msg)
	require.NoError(t, err)
	valid := k2.Verify(msg, signature)
	assert.True(t, valid)

	assert.NoError(t, err)
	assert.NotEmpty(t, pem1)
	assert.Equal(t, pem1, pem2)
}

func TestSaveLoadPubkey(t *testing.T) {
	k1 := keys.NewKey(keyType)
	err := k1.ExportPublicToFile(testPubKeyPemFile)
	assert.NoError(t, err)

	k2, err := keys.NewKeyFromFile(testPubKeyPemFile)
	require.NoError(t, err)
	require.NotEmpty(t, k2)
	pubEnc := k2.ExportPublic()
	assert.NotEmpty(t, pubEnc)
}

func TestSaveLoadPrivKeyNotFound(t *testing.T) {
	k1 := keys.NewKey(keyType)
	// no access
	err := k1.ExportPrivateToFile("/root")
	assert.Error(t, err)

	//
	err = k1.ImportPrivateFromFile("/filedoesnotexist.pem")
	assert.Error(t, err)
}

func TestSaveLoadPubKeyNotFound(t *testing.T) {
	k1 := keys.NewKey(keyType)
	err := k1.ImportPublicFromFile("/filedoesnotexist.pem")
	assert.Error(t, err)
}

func TestPublicKeyPEM(t *testing.T) {
	// golang generated public key (91 bytes after base64 decode) - THIS WORKS
	const TestKeyPub2 = "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEFFPcFfnGQr8/t2ZzWFYg/ZFLAkT0z/EYlC1RED4iot367KRNwZlilogTGHzi3HjH6NnL14d/DQHxAInctEeqxw=="
	const TestKeyPubPEM2 = "-----BEGIN PUBLIC KEY-----\nMFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAEaejQVxAbrUiN41Nqjgw8HG8q5OQM\nkveXku18/zhF2BbSfbQMnCSyP5VXCe/sgCEi62Qm0LYXd1VG2UQz38f4zQ==\n-----END PUBLIC KEY-----\n"

	// JS ellipsys generated public key (65 bytes after base64 decode) - THIS FAILS
	const TestJSEllipsysPub3 = "BKOVp2t2JLjodototsMvFbOJ1j9wTC4ITbOrnrb/EoJiQul9eoXmyHpaYnPztjPixFdiHk06NxGLDpxRDm5qXfo="

	// openssl generated public key - THIS WORKS
	const TestKeyOpenSSL = "MFkwEwYHKoZIzj0CAQYIKoZIzj0DAQcDQgAELtv253KXEWvWjCse0Wp5DprnXp5tp17C1Qtfjk5t/6+HPSc74uMQcp/KV++vc6OXJwk5XdZ8FSkUiU9cYRBo8A=="

	// JS elliptic encoded using base64 encoding of hex encoded pub key
	const TestKeyPub64hex = "MDRhMzk1YTc2Yjc2MjRiOGU4NzY4YjY4YjZjMzJmMTViMzg5ZDYzZjcwNGMyZTA4NGRiM2FiOWViNmZmMTI4MjYyNDJlOTdkN2E4NWU2Yzg3YTVhNjI3M2YzYjYzM2UyYzQ1NzYyMWU0ZDNhMzcxMThiMGU5YzUxMGU2ZTZhNWRmYQ=="

	// decode from base64 string. This succeeds. d2 is 91 bytes - this works
	k1 := keys.NewKeyFromEnc(TestKeyPub2)
	assert.NotNil(t, k1)
	assert.Equal(t, keys.KeyTypeECDSA, k1.KeyType())

	// decode from openssl generated public key. d2 is 91 bytes - this works
	k2 := keys.NewKeyFromEnc(TestKeyOpenSSL)
	assert.NotNil(t, k2)
	assert.Equal(t, keys.KeyTypeECDSA, k2.KeyType())

	// a hex key is not supported
	k3 := keys.NewKeyFromEnc(TestKeyPub64hex)
	assert.Nil(t, k3)

	////MarshalPKIXPublicKey converts a public key to PKIX, ASN.1 DER form
	//x509EncodedPub, err := x509.MarshalPKIXPublicKey(publicKey)
	//pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	// Parse JS elliptic generated key.
	// THIS FAILS as it is hex based without a DER prefix
	// d3 is 65 bytes which should be correct
	k4 := keys.NewKeyFromEnc(TestJSEllipsysPub3)
	assert.Nil(t, k4)
}

func TestPrivateKeyPEM(t *testing.T) {

	k1 := keys.NewKey(keyType)
	k1Enc := k1.ExportPrivate()
	assert.NotEmpty(t, k1Enc)

	k2 := keys.NewKey(keyType)
	err := k2.ImportPrivate(k1Enc)
	require.NoError(t, err)

	k2Enc := k2.ExportPrivate()
	require.NotNil(t, k2Enc)

	isEqual := k1Enc == k2Enc
	assert.True(t, isEqual)
}

func TestInvalidEnc(t *testing.T) {
	k1 := keys.NewKey(keyType)

	err := k1.ImportPrivate("PRIVATE KEY")
	assert.Error(t, err)

	// note: nkeys have not ability to verify the public key
	err = k1.ImportPublic("PUBLIC KEY")
	assert.Error(t, err)
}
