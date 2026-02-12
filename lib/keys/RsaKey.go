package keys

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"reflect"
)

// RsaKey implements the IHiveKeys interface to a RSA key.
type RsaKey struct {
	privKeyPtr crypto.PrivateKey // RSA native key type stored as pointer
	pubKeyPtr  crypto.PublicKey  // RSA native key type stored as pointer
}

// ExportPrivate returns the PEM encoded private key
func (k *RsaKey) ExportPrivate() string {
	var err error
	var pemEnc []byte
	var keyBytes []byte
	if k.privKeyPtr == nil {
		panic("private key not initialized")
	}

	keyBytes, err = x509.MarshalPKCS8PrivateKey(k.privKeyPtr)
	pemEnc = pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	if err != nil {
		panic("private key can't be marshalled: " + err.Error())
	}
	return string(pemEnc)
}

// ExportPrivateToFile saves the private key set to file in PEM format.
// The file permissions are set to 0400, current user only, read-write permissions.
//
//	Returns error in case the key is invalid or file cannot be written.
func (k *RsaKey) ExportPrivateToFile(pemPath string) error {
	privPEM := k.ExportPrivate()
	// remove existing key since perm 0400 doesn't allow overwriting it
	_ = os.Remove(pemPath)
	err := os.WriteFile(pemPath, []byte(privPEM), 0400)
	return err
}

// ExportPublic returns the PEM encoded public key if available
func (k *RsaKey) ExportPublic() (pemKey string) {
	var pemData []byte
	if k.pubKeyPtr == nil {
		panic("public key not initialized")
	}

	x509EncodedPub, err := x509.MarshalPKIXPublicKey(k.pubKeyPtr)
	if err != nil {
		panic("public key can't be marshalled")
	}
	pemData = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	if pemData == nil {
		panic("public key can't be marshalled")
	}
	return string(pemData)
}

// ExportPublicToFile saves the public key to file in PEM format.
// The file permissions are set to 0644, current user can write, rest can read.
//
//	Returns error in case the public key is invalid or file cannot be written.
func (k *RsaKey) ExportPublicToFile(pemPath string) error {
	pemEncoded := k.ExportPublic()
	err := os.WriteFile(pemPath, []byte(pemEncoded), 0644)
	return err
}

func (k *RsaKey) _importDer(privateEnc string) ([]byte, error) {
	var derBytes []byte
	var err error
	blockPub, _ := pem.Decode([]byte(privateEnc))
	if blockPub == nil {
		// not pem encoded. try base64
		//return fmt.Errorf("not a valid private key PEM string")
		derBytes, err = base64.StdEncoding.DecodeString(privateEnc)
	} else {
		derBytes = blockPub.Bytes
	}
	if err != nil {
		err = fmt.Errorf("ImportPrivate: unknown key format")
		return nil, err
	}
	return derBytes, nil
}

// ImportPrivate reads the key-pair from the PEM private key
// and determines its key type.
// This returns an error if the PEM is not a valid key.
func (k *RsaKey) ImportPrivate(privatePEM string) (err error) {
	derBytes, err := k._importDer(privatePEM)

	// this decodes RSA, ECDSA, ED25519 or ECDH key
	rawPrivateKey, err := x509.ParsePKCS8PrivateKey(derBytes)
	if err != nil {
		return err
	}
	privKey, valid := rawPrivateKey.(*rsa.PrivateKey)
	if !valid {
		keyType := reflect.TypeOf(k.pubKeyPtr)
		return fmt.Errorf("not an RSA private key. It looks to be a '%s'", keyType)
	}
	k.privKeyPtr = privKey
	k.pubKeyPtr = &privKey.PublicKey
	return err
}

// ImportPrivateFromFile loads public/private key pair from PEM file
// and determines its key type.
func (k *RsaKey) ImportPrivateFromFile(pemPath string) (err error) {
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
func (k *RsaKey) ImportPublic(publicPEM string) (err error) {
	derBytes, err := k._importDer(publicPEM)

	k.pubKeyPtr, err = x509.ParsePKIXPublicKey(derBytes)
	k.privKeyPtr = nil
	_, valid := k.pubKeyPtr.(*rsa.PublicKey)
	if !valid {
		keyType := reflect.TypeOf(k.pubKeyPtr)
		return fmt.Errorf("not an RSA public key. It looks to be a '%s'", keyType)
	}
	return err
}

// ImportPublicFromFile loads ECDSA public key from PEM file
func (k *RsaKey) ImportPublicFromFile(pemPath string) (err error) {
	pemEncodedPub, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPublic(string(pemEncodedPub))
	return err
}

// Initialize generates a new key
func (k *RsaKey) Initialize() IHiveKey {
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err.Error())
	}
	k.privKeyPtr = privKey
	k.pubKeyPtr = &privKey.PublicKey
	return k
}

// KeyType returns this key's type, eg ecdsa
func (k *RsaKey) KeyType() KeyType {
	return KeyTypeRSA
}

// PrivateKey returns the native *rsa.PrivateKey
func (k *RsaKey) PrivateKey() crypto.PrivateKey {
	return k.privKeyPtr
}

// PublicKey returns the native *rsa.PublicKey
func (k *RsaKey) PublicKey() crypto.PublicKey {
	return k.pubKeyPtr
}

// Sign returns the signature of a message signed using this key
// this requires a private key to be created or imported
func (k *RsaKey) Sign(msg []byte) (signature []byte, err error) {

	// https://www.sohamkamani.com/golang/rsa-encryption/
	// Before signing, we need to hash our message
	// The hash is what we actually sign
	msgHash := sha256.Sum256(msg)
	privKey := k.privKeyPtr.(*rsa.PrivateKey)
	signature, err = rsa.SignPKCS1v15(rand.Reader, privKey, crypto.SHA256, msgHash[:])
	if err != nil {
		log.Fatalf("Error signing message: %v", err)
	}
	return signature, err
}

// Verify the signature of a message using this key's public key
// this requires a public key to be created or imported
// returns true if the signature is valid for the message
func (k *RsaKey) Verify(msg []byte, signature []byte) (valid bool) {
	msgHash := sha256.Sum256(msg)
	pubKey := k.pubKeyPtr.(*rsa.PublicKey)
	err := rsa.VerifyPKCS1v15(pubKey, crypto.SHA256, msgHash[:], signature)
	valid = err == nil
	return valid
}

// NewRsaKey generates a RSA key with IHiveKey interface
func NewRsaKey() IHiveKey {
	k := &RsaKey{}
	k.Initialize()
	return k
}

// NewRsaKeyFromPrivate creates and initialize a IHiveKey object from an existing RSA private key.
func NewRsaKeyFromPrivate(privKey *rsa.PrivateKey) IHiveKey {
	k := &RsaKey{
		privKeyPtr: privKey,
		pubKeyPtr:  &privKey.PublicKey,
	}
	return k
}
