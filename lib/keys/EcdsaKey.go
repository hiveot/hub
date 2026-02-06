package keys

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"os"
	"reflect"
)

// EcdsaKey implements the secp256k ECDSA key.
// This implements the IHiveKeys interface.
type EcdsaKey struct {
	// Reuse RSA functionality
	RsaKey
}

// ImportPrivate reads the key-pair from the PEM private key
// and determines its key type.
// This returns an error if the PEM is not a valid key.
func (k *EcdsaKey) ImportPrivate(privatePEM string) (err error) {
	derBytes, err := k._importDer(privatePEM)
	_ = err
	rawPrivateKey, err := x509.ParsePKCS8PrivateKey(derBytes)
	if err != nil {
		return err
	}

	privKey, valid := rawPrivateKey.(*ecdsa.PrivateKey)
	if !valid {
		keyType := reflect.TypeOf(k.pubKeyPtr)
		return fmt.Errorf("not an ECDSA private key. It looks to be a '%s'", keyType)
	}
	k.privKeyPtr = privKey
	k.pubKeyPtr = &privKey.PublicKey

	return err
}

// ImportPrivateFromFile loads public/private key pair from PEM file
// and determines its key type.
func (k *EcdsaKey) ImportPrivateFromFile(pemPath string) (err error) {
	pemEncodedPriv, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPrivate(string(pemEncodedPriv))
	return err
}

// ImportPublic reads the public key from the PEM data.
// This returns an error if the PEM is not a valid public key
//
// publicPEM must contain either a PEM encoded string, or its base64 encoded content
func (k *EcdsaKey) ImportPublic(publicPEM string) (err error) {
	derBytes, err := k._importDer(publicPEM)
	_ = err
	k.pubKeyPtr, err = x509.ParsePKIXPublicKey(derBytes)
	k.privKeyPtr = nil
	_, valid := k.pubKeyPtr.(*ecdsa.PublicKey)
	if !valid {
		keyType := reflect.TypeOf(k.pubKeyPtr)
		return fmt.Errorf("not an ECDSA public key. It looks to be a '%s'", keyType)
	}
	return err
}

// ImportPublicFromFile loads ECDSA public key from PEM file
func (k *EcdsaKey) ImportPublicFromFile(pemPath string) (err error) {
	pemEncodedPub, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPublic(string(pemEncodedPub))
	return err
}

// Initialize generates a new key
func (k *EcdsaKey) Initialize() IHiveKey {
	curve := elliptic.P256()
	privKey, err := ecdsa.GenerateKey(curve, rand.Reader)
	if err != nil {
		panic("unable to create ECDSA key")
	}
	k.privKeyPtr = privKey
	k.pubKeyPtr = &privKey.PublicKey
	return k
}

// KeyType returns this key's type, eg ecdsa
func (k *EcdsaKey) KeyType() KeyType {
	return KeyTypeECDSA
}

// Sign returns the signature of a message signed using this key
// This signs the SHA256 hash of the message
// this requires a private key to be created or imported
func (k *EcdsaKey) Sign(msg []byte) (signature []byte, err error) {
	msgHash := sha256.Sum256(msg)
	privKey := k.privKeyPtr.(*ecdsa.PrivateKey)

	signature, err = ecdsa.SignASN1(rand.Reader, privKey, msgHash[:])
	return signature, err
}

// Verify the signature of a message using this key's public key
// This verifies using the SHA256 hash of the message.
// this requires a public key to be created or imported
// returns true if the signature is valid for the message
func (k *EcdsaKey) Verify(msg []byte, signature []byte) (valid bool) {
	msgHash := sha256.Sum256(msg)
	pubKey := k.pubKeyPtr.(*ecdsa.PublicKey)
	valid = ecdsa.VerifyASN1(pubKey, msgHash[:], signature)
	return valid
}

// NewEcdsaKey creates and initialize a ECDSA key
func NewEcdsaKey() IHiveKey {
	nk := &EcdsaKey{}
	nk.Initialize()
	return nk
}

// NewEcdsaKeyFromPrivate creates and initialize a IHiveKey object from an existing ECDSA private key.
func NewEcdsaKeyFromPrivate(privKey *ecdsa.PrivateKey) IHiveKey {
	k := &EcdsaKey{
		// reuse RSA
		RsaKey{
			privKeyPtr: privKey,
			pubKeyPtr:  &privKey.PublicKey,
		},
	}
	return k
}
