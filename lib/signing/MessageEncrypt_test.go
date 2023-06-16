package signing_test

import (
	"encoding/json"
	"testing"

	"github.com/hiveot/hub/lib/certsclient"
	"github.com/hiveot/hub/lib/signing"

	"github.com/stretchr/testify/assert"
)

// type TestObjectWithSender struct {
// 	Field1 string `json:"field1"`
// 	Field2 int    `json:"field2"`
// 	Sender string `json:"sender"`
// }
// type TestObjectNoSender struct {
// 	Address string `json:"address"`
// 	Field1  string `json:"field1"`
// 	Field2  int    `json:"field2"`
// 	// Sender string `json:"sender"`
// }

// const Pub1Address = "dom1.testpub.$identity"

// var testObject = TestObjectWithSender{
// 	Field1: "The question",
// 	Field2: 42,
// 	Sender: Pub1Address,
// }

// var testObject2 = TestObjectNoSender{
// 	Address: "test/publisher1/node1/input1/0/$set",
// 	Field1:  "The answer",
// 	Field2:  43,
// 	// Sender: Pub1Address,
// }

func TestEncryptDecrypt(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()
	const msg1 = "Message1"
	// encrypt using my own public key
	serialized, err := signing.EncryptMessage(msg1, &privKey.PublicKey)
	assert.NoError(t, err)

	// decrypt using my private key
	msg, isEncrypted, err := signing.DecryptMessage(serialized, privKey)
	assert.NoError(t, err)
	assert.True(t, isEncrypted)
	assert.Equal(t, msg1, msg)
}

func TestSignAndEncrypt(t *testing.T) {
	privKey := certsclient.CreateECDSAKeys()
	payload, _ := json.Marshal(testObject)

	msg, err := signing.SignAndEncrypt(payload, privKey, &privKey.PublicKey)
	_ = msg
	assert.NoError(t, err)
	// assert.NoErrorf(t,
}
