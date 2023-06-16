// Package certsclient with key management for clients (and server) certificates
package certsclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
)

// CreateECDSAKeys creates a asymmetric key set
// Clients save the private key locally, not to be shared with anyone and freely share
//
//	the public key. The keys are needed in client certificate creation.
//
// Returns a private key that contains its associated public key
func CreateECDSAKeys() *ecdsa.PrivateKey {
	rng := rand.Reader
	curve := elliptic.P256()
	privKey, _ := ecdsa.GenerateKey(curve, rng)
	return privKey
}

// LoadKeysFromPEM loads ECDSA public/private key pair from PEM file
func LoadKeysFromPEM(pemPath string) (privateKey *ecdsa.PrivateKey, err error) {
	pemEncodedPriv, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}
	return PrivateKeyFromPEM(string(pemEncodedPriv))
}

// LoadPublicKeyFromPEM loads ECDSA public key from file
func LoadPublicKeyFromPEM(pemPath string) (publicKey *ecdsa.PublicKey, err error) {
	pemEncodedKey, err := os.ReadFile(pemPath)
	if err != nil {
		return nil, err
	}
	return PublicKeyFromPEM(string(pemEncodedKey))
}

// PrivateKeyFromPEM converts a PEM encoded private key into a ECDSA private key object
// Intended to decode the public key portion of a certificate. This can be used to encrypt messages
// to the certificate holder.
func PrivateKeyFromPEM(pemEncodedKey string) (privateKey *ecdsa.PrivateKey, err error) {
	blockPub, _ := pem.Decode([]byte(pemEncodedKey))
	if blockPub == nil {
		return nil, errors.New("not a valid private key PEM string")
	}
	derBytes := blockPub.Bytes
	rawPrivateKey, err := x509.ParsePKCS8PrivateKey(derBytes)
	if err == nil {
		var ok bool
		privateKey, ok = rawPrivateKey.(*ecdsa.PrivateKey)
		if !ok || privateKey == nil {
			err = errors.New("PEM is not a ECDSA key format")
		}
	}
	return privateKey, err
}

// PublicKeyFromPEM converts a PEM encoded public key into a ECDSA or RSA public key object
// Intended to decode the public key portion of a certificate. This can be used to encrypt messages
// to the certificate holder.
func PublicKeyFromPEM(pemEncodedPub string) (publicKey *ecdsa.PublicKey, err error) {
	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	if blockPub == nil {
		return nil, errors.New("not a valid public key PEM string")
	}
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509EncodedPub)
	if err == nil {
		var ok bool
		publicKey, ok = genericPublicKey.(*ecdsa.PublicKey)
		if !ok || publicKey == nil {
			err = errors.New("not an ECDSA public key")
		}
	}

	return
}

// PrivateKeyToPEM converts the private/public key set to PEM formatted string.
// Returns error in case the private key is invalid
func PrivateKeyToPEM(privateKey interface{}) (string, error) {
	x509Encoded, err := x509.MarshalPKCS8PrivateKey(privateKey)
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})
	return string(pemEncoded), err
}

// PublicKeyToPEM converts a public key into PEM encoded format.
// Intended to send someone the public key in a transmissible format.
// See also PublicKeyFromPem for its counterpart
//
//	publicKey is the *rsa.PublicKey, *ecdsa.PublicKey or edd25519.PublicKey
func PublicKeyToPEM(publicKey interface{}) (string, error) {
	x509EncodedPub, err := x509.MarshalPKIXPublicKey(publicKey)
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	return string(pemEncodedPub), err
}

// SaveKeysToPEM saves the private/public key set to file in PEM format.
// The file permissions are set to 0600, current user only, read-write permissions.
//
//	privateKey is the *rsa.PrivateKey, *ecdsa.PrivateKey, or *edd25519.PrivateKey
//	Returns error in case the key is invalid or file cannot be written.
func SaveKeysToPEM(privateKey interface{}, pemPath string) error {
	x509Encoded, err := x509.MarshalPKCS8PrivateKey(privateKey)
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})
	if err == nil {
		err = os.WriteFile(pemPath, pemEncoded, 0600)
	}
	return err
}

// SavePublicKeyToPEM saves the public key to file in PEM format.
// The file permissions are set to 0644, current user can write, rest can read.
//
//	publicKey is the *rsa.PublicKey, *ecdsa.PublicKey or edd25519.PublicKey
//	Returns error in case the public key is invalid or file cannot be written.
func SavePublicKeyToPEM(pubKey interface{}, pemPath string) error {
	pemEncoded, err := PublicKeyToPEM(pubKey)
	if err == nil {
		err = os.WriteFile(pemPath, []byte(pemEncoded), 0644)
	}
	return err
}
