package signing_test

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/json"
	"log"
	"testing"
	"time"

	"gopkg.in/square/go-jose.v2"

	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/signing"

	"github.com/stretchr/testify/assert"
)

type TestObjectWithSender struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
	Sender string `json:"sender"`
}
type TestObjectNoSender struct {
	Address string `json:"address"`
	Field1  string `json:"field1"`
	Field2  int    `json:"field2"`
	// Sender string `json:"sender"`
}

const Pub1Address = "dom1.testpub.$identity"

var testObject = TestObjectWithSender{
	Field1: "The question",
	Field2: 42,
	Sender: Pub1Address,
}

var testObject2 = TestObjectNoSender{
	Address: "test/publisher1/node1/input1/0/$set",
	Field1:  "The answer",
	Field2:  43,
	// Sender: Pub1Address,
}

func TestEcdsaSigning(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()
	payload, _ := json.Marshal(testObject)

	sig := signing.CreateEcdsaSignature(payload, privKey)
	err := signing.VerifyEcdsaSignature(payload, sig, &privKey.PublicKey)
	assert.NoErrorf(t, err, "Verification of ECDSA signature failed")

	// error cases - test without pk
	sig = signing.CreateEcdsaSignature(payload, nil)
	assert.Empty(t, sig, "Expected no signature without keys")
	// test with invalid payload
	err = signing.VerifyEcdsaSignature([]byte("hello world"), sig, &privKey.PublicKey)
	assert.Error(t, err)
	// test with invalid signature
	err = signing.VerifyEcdsaSignature(payload, "invalid sig", &privKey.PublicKey)
	assert.Error(t, err)
	// test with invalid public key
	sig = signing.CreateEcdsaSignature(payload, privKey)
	err = signing.VerifyEcdsaSignature(payload, sig, nil)
	assert.Error(t, err)
	newKey := certsclient.CreateECDSAKeys()
	err = signing.VerifyEcdsaSignature(payload, sig, &newKey.PublicKey)
	assert.Error(t, err)

}

func TestJWSSigning(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()

	payload1, err := json.Marshal(testObject)
	assert.NoErrorf(t, err, "Serializing node1 failed")

	sig1, err := signing.CreateJWSSignature(payload1, privKey)
	assert.NoErrorf(t, err, "signing node1 failed")
	assert.NotEmpty(t, sig1, "Signature is empty")

	sig2 := signing.CreateEcdsaSignature(payload1, privKey)
	assert.NotEqual(t, sig1, sig2, "JWS Signature doesn't match with Ecdsa")

}

func TestSigningPerformance(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()

	payload1, err := json.Marshal(testObject)
	assert.NoErrorf(t, err, "Serializing node1 failed")

	// create sig of base64URL encoded publisher
	start := time.Now()
	for count := 0; count < 10000; count++ {
		payload1base64 := base64.URLEncoding.EncodeToString(payload1)
		sig := signing.CreateEcdsaSignature([]byte(payload1base64), privKey)
		_ = sig
	}
	duration := time.Since(start).Seconds()
	log.Printf("10K CreateEcdsaSignature signatures generated in %f seconds", duration)

	// Create JWS signature using JOSE directly
	start = time.Now()
	joseSigner, _ := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.ES256, Key: privKey}, nil)
	for count := 0; count < 10000; count++ {
		jws, _ := joseSigner.Sign([]byte(payload1))
		payload, _ := jws.CompactSerialize()
		_ = payload
	}
	duration = time.Since(start).Seconds()
	log.Printf("10K JoseSigner signatures generated in %.2f seconds", duration)

	// generate JWS signature using my lib
	start = time.Now()
	for count := 0; count < 10000; count++ {
		_, _ = signing.CreateJWSSignature(payload1, privKey)
	}
	duration = time.Since(start).Seconds()
	log.Printf("10K CreateJWSSignature signatures generated in %.2f seconds", duration)

	// verify sig of base64URL encoded payload
	payload1base64 := base64.URLEncoding.EncodeToString(payload1)
	sig := signing.CreateEcdsaSignature([]byte(payload1base64), privKey)
	start = time.Now()
	for count := 0; count < 10000; count++ {
		// verify signature
		// pubKey2 = messenger.PublicKeyFromPem(pubPem)
		match := signing.VerifyEcdsaSignature([]byte(payload1base64), sig, &privKey.PublicKey)
		_ = match
	}
	duration = time.Since(start).Seconds()
	log.Printf("10K VerifyEcdsaSignature signatures verified in %f seconds", duration)

	// verify JWS signature using my lib
	sig1, err := signing.CreateJWSSignature(payload1, privKey)
	assert.NoError(t, err)
	start = time.Now()
	for count := 0; count < 10000; count++ {
		_, _ = signing.VerifyJWSMessage(sig1, &privKey.PublicKey)
	}
	duration = time.Since(start).Seconds()
	log.Printf("10K VerifyJWSMessage signatures verified in %.2f seconds", duration)
}

// Test the sender verification
func TestVerifySender(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()

	payload1, err := json.Marshal(testObject)
	assert.NoErrorf(t, err, "Serializing node1 failed")
	assert.NoError(t, err)
	sig1, err := signing.CreateJWSSignature(payload1, privKey)
	assert.NoError(t, err)

	var received TestObjectWithSender
	isSigned, err := signing.VerifySenderJWSSignature(sig1, &received, func(address string) *ecdsa.PublicKey {
		// return the public key of this publisher
		return &privKey.PublicKey
	})
	assert.NoErrorf(t, err, "Verification failed")
	assert.True(t, isSigned, "Message wasn't signed")

	// using 'Address' instead of sender in the payload
	payload2, err := json.Marshal(testObject2)
	assert.NoError(t, err)
	sig2, err := signing.CreateJWSSignature(payload2, privKey)
	assert.NoError(t, err)
	var received2 TestObjectNoSender
	isSigned, err = signing.VerifySenderJWSSignature(sig2, &received2, func(address string) *ecdsa.PublicKey {
		// return the public key of this publisher
		return &privKey.PublicKey
	})
	assert.NoErrorf(t, err, "Verification failed")
	assert.True(t, isSigned, "Message wasn't signed")

	// using no public key lookup
	payload2, err = json.Marshal(testObject2)
	assert.NoError(t, err)
	sig2, err = signing.CreateJWSSignature(payload2, privKey)
	assert.NoError(t, err)
	isSigned, err = signing.VerifySenderJWSSignature(sig2, &received2, nil)
	assert.NoErrorf(t, err, "Verification without public key lookup function should succeed")
	assert.True(t, isSigned, "Message wasn't signed")

	// no public key for sender
	sig2, err = signing.CreateJWSSignature(payload2, privKey)
	assert.NoError(t, err)
	isSigned, err = signing.VerifySenderJWSSignature(sig2, &received2, func(address string) *ecdsa.PublicKey {
		return nil
	})
	assert.Errorf(t, err, "Verification without public key succeeded")
	assert.True(t, isSigned, "Message wasn't signed")

	// using empty address
	testObject2.Address = ""
	payload2, err = json.Marshal(testObject2)
	assert.NoError(t, err)
	sig2, err = signing.CreateJWSSignature(payload2, privKey)
	assert.NoError(t, err)
	isSigned, err = signing.VerifySenderJWSSignature(sig2, &received2, nil)
	assert.Errorf(t, err, "Verification with message without Address should not succeed")
	assert.True(t, isSigned, "Message wasn't signed")

	// no sender or address
	var obj3 struct{ Field1 string }
	payload3, err := json.Marshal(obj3)
	assert.NoError(t, err)
	sig3, err := signing.CreateJWSSignature(payload3, privKey)
	assert.NoError(t, err)
	isSigned, err = signing.VerifySenderJWSSignature(sig3, &obj3, nil)
	assert.Errorf(t, err, "Verification with message without sender should not succeed")
	assert.True(t, isSigned, "Message wasn't signed")

	// invalid message
	_, err = signing.VerifySenderJWSSignature("invalid", &received, nil)
	assert.Errorf(t, err, "Verification with invalid message should not succeed")
	// invalid payload
	payload4 := []byte("this is not json")
	sig4, err := signing.CreateJWSSignature(payload4, privKey)
	assert.NoError(t, err)
	_, err = signing.VerifySenderJWSSignature(sig4, &received, nil)
	assert.Errorf(t, err, "Verification with non json payload should not succeed")

	// different public key
	newKeys := certsclient.CreateECDSAKeys()
	_, err = signing.VerifySenderJWSSignature(sig2, &received, func(address string) *ecdsa.PublicKey {
		// return the public key of this publisher
		return &newKeys.PublicKey
	})
	assert.Errorf(t, err, "Verification with wrong publickey should not succeed")

	// error case - verify invalid jws mess
	_, err = signing.VerifyJWSMessage("bad sig", &privKey.PublicKey)
	assert.Error(t, err, "Invalid sign should result in error")

	_, err = signing.VerifyJWSMessage(sig1, nil)
	assert.Error(t, err, "nil public key should result in error")
}
