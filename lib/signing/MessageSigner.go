// Package messaging for signing and encryption of messages
package signing

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"encoding/asn1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"

	"gopkg.in/square/go-jose.v2"
)

// !!! THIS CODE IS NOT YET IN USE !!!
// The message envelope is used if a message is signed
type MessageSignatureEnvelope struct {
	Sender    string `json:"sender"`    // sender clientID
	Signature []byte `json:"signature"` // base64 encoded signature
	Payload   []byte `json:"payload"`   // base64 encoded payload
}

// ECDSASignature ...
type ECDSASignature struct {
	R, S *big.Int
}

// MessageSigner for signing and verifying of signed and encrypted messages using ECDSA
type MessageSigner struct {
	// GetPublicKey when available is used in message to verify signature
	GetPublicKey func(address string) *ecdsa.PublicKey // must be a variable
	// messenger    IMessenger
	privateKey *ecdsa.PrivateKey // private key for signing and decryption.
}

// DecodeMessage decrypts the message and verifies the sender signature.
// The sender and signer of the message is contained the message 'sender' field. If the
// Sender field is missing then the 'address' field is used as sender.
//
//	rawMessage contains the encryped and signed message
//	object must hold the expected message type to decode the json message
func (signer *MessageSigner) DecodeMessage(rawMessage string, object interface{}) (isEncrypted bool, isSigned bool, err error) {

	dmessage, isEncrypted, err := DecryptMessage(rawMessage, signer.privateKey)
	if err != nil {
		return false, false, err
	}
	isSigned, err = VerifySenderJWSSignature(dmessage, object, signer.GetPublicKey)
	return isEncrypted, isSigned, err
}

// VerifySignedMessage parses and verifies the message signature
// as per standard, the sender and signer of the message is in the message 'Sender' field. If the
// Sender field is missing then the 'address' field contains the publisher.
//
//	rawMessage contains the signed message
//	object must hold the expected message type to decode the json message
func (signer *MessageSigner) VerifySignedMessage(rawMessage string, object interface{}) (isSigned bool, err error) {
	isSigned, err = VerifySenderJWSSignature(rawMessage, object, signer.GetPublicKey)
	return isSigned, err
}

// CreateEcdsaSignature creates a ECDSA256 signature from the payload using the provided private key
// This returns a base64url encoded signature
//
//	payload to create the signature for
//	privateKey used to sign. The receiver must have the public key to verify the signature
func CreateEcdsaSignature(payload []byte, privateKey *ecdsa.PrivateKey) string {
	if privateKey == nil {
		return ""
	}
	hashed := sha256.Sum256(payload)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hashed[:])
	if err != nil {
		return ""
	}
	sig, err := asn1.Marshal(ECDSASignature{r, s})
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(sig)
}

// CreateJWSSignature signs the payload using JSE ES256 and return the JSE compact serialized message
//
//	payload to create the signature for and serialize
//	privateKey used to sign. The received must have the public key to verify
//
// This returns the JSE compact serialized message
func CreateJWSSignature(payload []byte, privateKey *ecdsa.PrivateKey) (string, error) {
	joseSigner, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.ES256, Key: privateKey}, nil)
	if err != nil {
		return "", err
	}
	signedObject, err := joseSigner.Sign(payload)
	if err != nil {
		return "", err
	}
	// serialized := signedObject.FullSerialize()
	serialized, err := signedObject.CompactSerialize()
	return serialized, err
}

// VerifyEcdsaSignature the payload using the base64url encoded signature and public key
// payload is any raw data
// signatureB64urlEncoded is the ecdsa 256 URL encoded signature
// Intended for signing an object like the publisher identity. Use VerifyJWSMessage for
// verifying JWS signed messages.
func VerifyEcdsaSignature(payload []byte, signatureB64urlEncoded string, publicKey *ecdsa.PublicKey) error {
	var rs ECDSASignature
	if publicKey == nil {
		return errors.New("VerifyEcdsaSignature: publicKey is nil")
	}
	signature, err := base64.URLEncoding.DecodeString(signatureB64urlEncoded)
	if err != nil {
		return errors.New("VerifyEcdsaSignature: Invalid signature")
	}

	if _, err = asn1.Unmarshal(signature, &rs); err != nil {
		return errors.New("VerifyEcdsaSignature: Payload is not ASN")
	}

	hashed := sha256.Sum256(payload)
	verified := ecdsa.Verify(publicKey, hashed[:], rs.R, rs.S)
	if !verified {
		return errors.New("VerifyEcdsaSignature: Signature does not match payload")
	}
	return nil
}

// VerifyJWSMessage verifies a signed message and returns its payload
// The message is a JWS encoded string. The public key of the sender is
// needed to verify the message.
//
//	Intended for testing, as the application uses VerifySenderJWSSignature instead.
func VerifyJWSMessage(message string, publicKey *ecdsa.PublicKey) (payload string, err error) {
	if publicKey == nil {
		err := errors.New("VerifyJWSMessage: public key is nil")
		return "", err
	}
	jwsSignature, err := jose.ParseSigned(message)
	if err != nil {
		return "", err
	}
	payloadB, err := jwsSignature.Verify(publicKey)
	return string(payloadB), err
}

// VerifySenderJWSSignature verifies if a message is JWS signed. If signed then the signature is verified
// using the 'Sender' or 'Address' attributes to determine the public key to verify with.
// To verify correctly, the sender has to be a known publisher and verified with the DSS.
//
//	object MUST be a pointer to the type otherwise unmarshal fails.
//
// getPublicKey is a lookup function for providing the public key from the given sender address.
//
//	it should only provide a public key if the publisher is known and verified by the DSS, or
//	if this zone does not use a DSS (publisher are protected through message bus ACLs)
//	If not provided then signature verification will succeed.
//
// The rawMessage is json unmarshalled into the given object.
//
// This returns a flag if the message was signed and if so, an error if the verification failed
func VerifySenderJWSSignature(rawMessage string, object interface{}, getPublicKey func(address string) *ecdsa.PublicKey) (isSigned bool, err error) {

	jwsSignature, err := jose.ParseSigned(rawMessage)
	if err != nil {
		// message is (probably) not signed, try to unmarshal it directly
		err = json.Unmarshal([]byte(rawMessage), object)
		return false, err
	}
	payload := jwsSignature.UnsafePayloadWithoutVerification()
	err = json.Unmarshal([]byte(payload), object)
	if err != nil {
		// message doesn't have a json payload
		errTxt := fmt.Sprintf("VerifySenderSignature: Signature okay but message unmarshal failed: %s", err)
		return true, errors.New(errTxt)
	}
	// determine who the sender is
	reflObject := reflect.ValueOf(object).Elem()
	reflSender := reflObject.FieldByName("Sender")
	if !reflSender.IsValid() {
		reflSender = reflObject.FieldByName("Address")
		if !reflSender.IsValid() {
			err = errors.New("VerifySenderJWSSignature: object doesn't have a Sender or Address field")
			return true, err
		}
	}
	sender := reflSender.String()
	if sender == "" {
		err := errors.New("VerifySenderJWSSignature: Missing sender or address information in message")
		return true, err
	}
	// verify the message signature using the sender's public key
	if getPublicKey == nil {
		return true, nil
	}
	publicKey := getPublicKey(sender)
	if publicKey == nil {
		err := errors.New("VerifySenderJWSSignature: No public key available for sender " + sender)
		return true, err
	}

	_, err = jwsSignature.Verify(publicKey)
	if err != nil {
		msg := fmt.Sprintf("VerifySenderJWSSignature: message signature from %s fails to verify with its public key", sender)
		err := errors.New(msg)
		return true, err
	}
	return true, err
}
