// Package keys with key generation
package keys

import "crypto"

type KeyType string

const (
	KeyTypeECDSA   KeyType = "ecdsa"
	KeyTypeEd25519 KeyType = "ed25519"
	KeyTypeRSA     KeyType = "rsa"
	KeyTypeNKey    KeyType = "nkey"
	KeyTypeUnknown         = ""
)

// KPFileExt defines the filename extension under which public/private keys are stored
// in the keys directory.
const KPFileExt = ".key"

// PubKeyFileExt defines the filename extension under which public key is stored
// in the keys directory.
const PubKeyFileExt = ".pub"

// IHiveKey defines the standard interface for various key types used for signing and authentication
//
// ... because we don't care about all these keys, just that it works and is secure...
type IHiveKey interface {

	// ExportPrivate returns the serialized private key if available
	// This defaults to PEM encoding unless the key type doesn't support it.
	//  key type ecdsa, rsa use PEM encoding
	//  key type ed25519 encodes it to base64
	//  key type nkeys encodes when generating its seed
	ExportPrivate() string

	// ExportPrivateToFile saves the private/public key to a key file
	ExportPrivateToFile(keyPath string) error

	// ExportPublic returns the serialized public key if available
	// This defaults to PEM encoding unless the key type doesn't support it.
	ExportPublic() string

	// ExportPublicToFile exports the public key and write to file.
	// This defaults to PEM encoding unless the key type doesn't support it.
	ExportPublicToFile(pemPath string) error

	// ImportPrivate decodes the key-pair from the serialized private key
	// This returns an error if the encoding can't be determined
	ImportPrivate(privateEnc string) error

	// ImportPrivateFromFile private/public key from a PEM file
	ImportPrivateFromFile(pemPath string) error

	// ImportPublic reads the public key from the given encoded data.
	// Intended for verifying signatures using the public key.
	// This returns an error if the encoding can't be determined
	ImportPublic(publicEnc string) error

	// ImportPublicFromFile reads the public key from file.
	// The encoding depends on the key type. ed25519, ecdsa and rsa uses pem format.
	// Intended for verifying signatures using the public key.
	// This returns an error if the file cannot be read or is not a valid public key
	// Note that after ImportPublicFrom...(), the private key is not available.
	ImportPublicFromFile(pemPath string) error

	// Initialize generates a new key set using its curve algorithm
	Initialize() IHiveKey

	// KeyType returns the key's type
	KeyType() KeyType

	// PrivateKey returns the native private key
	//	ECDSA:   *ecdsa.PrivateKey
	//	RSA:     *rsa.PrivateKey
	//	ED25519: ed25519.PrivateKey (not a pointer)
	//	nkeys:   nkeys.KeyPair
	PrivateKey() crypto.PrivateKey

	// PublicKey returns the native public key
	//	ECDSA:   *ecdsa.PublicKey
	//	RSA:     *rsa.PublicKey
	//	ED25519: ed25519.PublicKey (not a pointer)
	//	nkeys:   nkeys.KeyPair
	PublicKey() crypto.PublicKey

	// Sign returns the signature of a message signed using this key
	// this requires a private key to be created or imported
	Sign(message []byte) ([]byte, error)

	// Verify the message signature using this key's public key
	// this requires a public key to be created or imported
	// returns true if the signature is valid for the message
	Verify(message []byte, signature []byte) bool
}
