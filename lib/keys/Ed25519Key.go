package keys

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"os"
)

// Ed25519Key contains the ED25519 cryptographic key set for signing and authentication.
// This implements the IHiveKeys interface.
type Ed25519Key struct {
	privKey ed25519.PrivateKey // []byte key type
	pubKey  ed25519.PublicKey  // []byte ley type
}

// ExportPrivate returns the PEM encoded private key
func (k *Ed25519Key) ExportPrivate() string {
	var pemEnc []byte
	if k.privKey == nil {
		panic("private key not initialized")
	}
	raw, err := x509.MarshalPKCS8PrivateKey(k.privKey)
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: raw,
	}
	pemEnc = pem.EncodeToMemory(block)
	_ = err
	return string(pemEnc)
}

// ExportPrivateToFile saves the private key set to file in PEM format.
// The file permissions are set to 0400, current user only, read-write permissions.
//
//	Returns error in case the key is invalid or file cannot be written.
func (k *Ed25519Key) ExportPrivateToFile(pemPath string) error {
	privPEM := k.ExportPrivate()
	// remove existing key since perm 0400 doesn't allow overwriting it
	_ = os.Remove(pemPath)
	err := os.WriteFile(pemPath, []byte(privPEM), 0400)
	return err
}

// ExportPublicToFile saves the public key to file in PEM format.
// The file permissions are set to 0644, current user can write, rest can read.
//
//	Returns error in case the public key is invalid or file cannot be written.
func (k *Ed25519Key) ExportPublicToFile(pemPath string) error {
	pemEncoded := k.ExportPublic()
	err := os.WriteFile(pemPath, []byte(pemEncoded), 0644)
	return err
}

// ExportPublic returns the PEM encoded public key if available
func (k *Ed25519Key) ExportPublic() (pemKey string) {
	var pemData []byte
	if k.pubKey == nil {
		panic("public key not initialized")
	}

	x509EncodedPub, err := x509.MarshalPKIXPublicKey(k.pubKey)
	if err != nil {
		panic("ED25519 public key can't be marshalled")
	}
	pemData = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})
	return string(pemData)
}

// ImportPrivateFromFile loads public/private key pair from PEM file
// and determines its key type.
func (k *Ed25519Key) ImportPrivateFromFile(pemPath string) (err error) {
	pemEncodedPriv, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPrivate(string(pemEncodedPriv))
	return err
}

// ImportPrivate reads the key-pair from PEM format.
// This returns an error if the PEM is not a valid key.
func (k *Ed25519Key) ImportPrivate(privatePEM string) (err error) {
	var derBytes []byte
	blockPub, _ := pem.Decode([]byte(privatePEM))
	if blockPub == nil {
		// not pem encoded. try base64
		derBytes, err = base64.StdEncoding.DecodeString(privatePEM)
	} else {
		derBytes = blockPub.Bytes
	}
	if err != nil {
		err = fmt.Errorf("not a PEM or base64 encoded format")
		return err
	}
	// try PKCS8 encoding
	rawPrivateKey, err := x509.ParsePKCS8PrivateKey(derBytes)
	ed25519PK, found := rawPrivateKey.(ed25519.PrivateKey)
	if found {
		k.privKey = ed25519PK
		k.pubKey = k.privKey.Public().(ed25519.PublicKey)
		return nil
	}

	// is it a ed25519 seed?
	if len(derBytes) != ed25519.SeedSize {
		err = fmt.Errorf("not a ED25519 seed")
		return err
	}
	k.privKey = ed25519.NewKeyFromSeed(derBytes)
	k.pubKey = k.privKey.Public().(ed25519.PublicKey)
	return err
}

// ImportPublic reads the public key from the PEM data.
// This returns an error if the PEM is not a valid public key
//
// publicPEM must contain either a PEM encoded string, or its base64 encoded content
func (k *Ed25519Key) ImportPublic(publicPEM string) (err error) {
	var x509EncodedPub []byte
	blockPub, _ := pem.Decode([]byte(publicPEM))
	if blockPub == nil {
		// try just base64 decoding
		x509EncodedPub, err = base64.StdEncoding.DecodeString(publicPEM)
	} else {
		x509EncodedPub = blockPub.Bytes
	}
	if err != nil {
		err = fmt.Errorf("ImportPublic: not an ED25519 public key format")
		return err
	}
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509EncodedPub)
	if err != nil {
		return err
	}
	k.pubKey = genericPublicKey.(ed25519.PublicKey)
	k.privKey = nil
	return err
}

// ImportPublicFromFile loads ED25519 public key from PEM file
func (k *Ed25519Key) ImportPublicFromFile(pemPath string) (err error) {
	pemEncodedPub, err := os.ReadFile(pemPath)
	if err != nil {
		return err
	}
	err = k.ImportPublic(string(pemEncodedPub))
	return err
}

// Initialize generates a new key
func (k *Ed25519Key) Initialize() IHiveKey {
	var err error
	k.pubKey, k.privKey, err = ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err.Error())
	}
	return k
}

// KeyType returns this key's type, eg ED25519
func (k *Ed25519Key) KeyType() KeyType {
	return KeyTypeEd25519
}

// PrivateKey returns the native private key pointer
func (k *Ed25519Key) PrivateKey() crypto.PrivateKey {
	return k.privKey
}

// PublicKey returns the native public key pointer
func (k *Ed25519Key) PublicKey() crypto.PublicKey {
	return k.pubKey
}

// Sign returns the signature of a message signed using this key
// This signs the SHA256 hash of the message
// this requires a private key to be created or imported
func (k *Ed25519Key) Sign(msg []byte) (signature []byte, err error) {
	msgHash := sha256.Sum256(msg)
	signature = ed25519.Sign(k.privKey, msgHash[:])
	return signature, nil
}

// Verify the signature of a message using this key's public key.
// This verifies using the SHA256 hash of the message.
// this requires a public key to be created or imported
// returns true if the signature is valid for the message
func (k *Ed25519Key) Verify(msg []byte, signature []byte) (valid bool) {
	msgHash := sha256.Sum256(msg)
	valid = ed25519.Verify(k.pubKey, msgHash[:], signature)
	return valid
}

// NewEd25519Key creates and initialize a ED25519 key
func NewEd25519Key() IHiveKey {
	k := &Ed25519Key{}
	k.Initialize()
	return k
}

// NewEd25519KeyFromPrivate creates and initialize a IHiveKey object from an existing private key.
func NewEd25519KeyFromPrivate(privKey ed25519.PrivateKey) IHiveKey {
	pubKey := privKey.Public()
	k := &Ed25519Key{
		privKey: privKey,
		pubKey:  pubKey.(ed25519.PublicKey),
	}
	return k
}
