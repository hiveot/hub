package keys

import (
	"crypto"
	"fmt"
	"github.com/nats-io/nkeys"
	"os"
)

// NKeysKey contains an asymmetric cryptographic key set for signing and authentication.
// This implements the IHiveKeys interface.
type NKeysKey struct {
	privKey crypto.PrivateKey // pointer key type
	pubKey  crypto.PublicKey  // pointer key type
}

// ExportPrivate returns the encoded private key
func (k *NKeysKey) ExportPrivate() string {
	var err error
	var privEnc []byte
	if k.privKey == nil {
		panic("private key not initialized")
	}

	// export as decorated key (eg, the nkey seed)
	kp := k.privKey.(nkeys.KeyPair)
	privEnc, err = kp.Seed()
	if err != nil {
		panic("private key can't be marshalled: " + err.Error())
	}
	return string(privEnc)
}

// ExportPrivateToFile saves the private key set to file in PEM format.
// The file permissions are set to 0400, current user only, read-write permissions.
//
//	Returns error in case the key is invalid or file cannot be written.
func (k *NKeysKey) ExportPrivateToFile(privPath string) error {
	privEnc := k.ExportPrivate()
	// remove existing key since perm 0400 doesn't allow overwriting it
	_ = os.Remove(privPath)
	err := os.WriteFile(privPath, []byte(privEnc), 0400)
	return err
}

// ExportPublic returns the encoded public key if available
func (k *NKeysKey) ExportPublic() (pubEnc string) {
	if k.pubKey == nil {
		panic("public key not initialized")
	}

	// nkeys directly provide a base64 encoded key
	pubEnc = *k.pubKey.(*string)
	return string(pubEnc)
}

// ExportPublicToFile saves the public key to file in PEM format.
// The file permissions are set to 0644, current user can write, rest can read.
//
//	Returns error in case the public key is invalid or file cannot be written.
func (k *NKeysKey) ExportPublicToFile(pubPath string) error {
	pubEnc := k.ExportPublic()
	err := os.WriteFile(pubPath, []byte(pubEnc), 0644)
	return err
}

// ImportPrivate reads the encoded private nkey
func (k *NKeysKey) ImportPrivate(privatePEM string) error {
	nk, err := nkeys.ParseDecoratedUserNKey([]byte(privatePEM))
	if err != nil {
		err = fmt.Errorf("unknown key format")
		return err
	}

	// to exit the flow right here as x509.ParsePKCS8PrivateKey won't work
	k.privKey = nk
	pubKey, _ := nk.PublicKey()
	k.pubKey = &pubKey

	return nil
}

// ImportPrivateFromFile loads public/private key pair from PEM file
// and determines its key type.
func (k *NKeysKey) ImportPrivateFromFile(pemPath string) (err error) {
	privEnc, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPrivate(string(privEnc))
	return err
}

// ImportPublic reads the public key from the encoded data.
// Note that nkeys need a key-pair for authentication, signing and verification
func (k *NKeysKey) ImportPublic(pubEnc string) (err error) {
	// the public key remains in its encoded format
	// how to determine if this is a valid public key?
	// a keypair is needed for signing and verification,
	k.pubKey = &pubEnc
	return err
}

// ImportPublicFromFile loads ECDSA public key from PEM file
func (k *NKeysKey) ImportPublicFromFile(pemPath string) (err error) {
	pemEncodedPub, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPublic(string(pemEncodedPub))
	return err
}

// Initialize initializes a new key set
func (k *NKeysKey) Initialize() IHiveKey {
	kp, err := nkeys.CreateUser()
	if err != nil {
		panic(err.Error())
	}
	k.privKey = kp
	pubKey, _ := kp.PublicKey()
	k.pubKey = &pubKey // pointer

	return k
}

// KeyType returns this key's type, eg ecdsa
func (k *NKeysKey) KeyType() KeyType {
	return KeyTypeNKey
}

// PrivateKey returns the native nkey.KeyPair private key
func (k *NKeysKey) PrivateKey() crypto.PrivateKey {
	return k.privKey
}

// PublicKey returns the native public key
func (k *NKeysKey) PublicKey() crypto.PublicKey {
	return k.pubKey
}

// Sign returns the signature of a message signed using this key
// this requires a private key to be created or imported
func (k *NKeysKey) Sign(msg []byte) (signature []byte, err error) {
	kp := k.privKey.(nkeys.KeyPair)
	signature, err = kp.Sign(msg)
	return signature, err
}

// Verify the signature of a message using this key's public key
// this requires a public key to be created or imported
// returns true if the signature is valid for the message
func (k *NKeysKey) Verify(msg []byte, signature []byte) (valid bool) {
	kp := k.privKey.(nkeys.KeyPair)
	err := kp.Verify(msg, signature)
	return err == nil
}

// NewNkeysKey creates and initialize a new nkey key
func NewNkeysKey() IHiveKey {
	k := &NKeysKey{}
	return k.Initialize()
}

// NewNKeysKeyFromPrivate creates a new key set from an existing nkey keypair
func NewNKeysKeyFromPrivate(privKey nkeys.KeyPair) IHiveKey {
	pubKey, _ := privKey.PublicKey()
	k := &NKeysKey{
		privKey: privKey,
		pubKey:  pubKey,
	}
	return k
}
