package signing

import (
	"crypto/ecdsa"

	"gopkg.in/square/go-jose.v2"
)

// DecryptMessage deserializes and decrypts the message using JWE
// This returns the decrypted message, or the input message if the message was not encrypted
func DecryptMessage(serialized string, privateKey *ecdsa.PrivateKey) (message string, isEncrypted bool, err error) {
	message = serialized
	decrypter, err := jose.ParseEncrypted(serialized)
	if err == nil {
		dmessage, err := decrypter.Decrypt(privateKey)
		message = string(dmessage)
		return message, true, err
	}
	return message, false, err
}

// EncryptMessage encrypts and serializes the message using JWE
func EncryptMessage(message string, publicKey *ecdsa.PublicKey) (serialized string, err error) {
	var jwe *jose.JSONWebEncryption

	recpnt := jose.Recipient{Algorithm: jose.ECDH_ES, Key: publicKey}

	encrypter, err := jose.NewEncrypter(jose.A128CBC_HS256, recpnt, nil)

	if encrypter != nil {
		jwe, err = encrypter.Encrypt([]byte(message))
	}
	if err != nil {
		return message, err
	}
	serialized, _ = jwe.CompactSerialize()
	return serialized, err
}

// Encrypt signs and encrypts the payload
// This returns the JWS signed and JWE encrypted message
func SignAndEncrypt(payload []byte, myPrivateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey) (message string, err error) {
	// first sign, then encrypt as per RFC
	message, err = CreateJWSSignature(payload, myPrivateKey)
	if err != nil {
		return "", err
	}
	emessage, err := EncryptMessage(message, publicKey)
	return emessage, err
}
