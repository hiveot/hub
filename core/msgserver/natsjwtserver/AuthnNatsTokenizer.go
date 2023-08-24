package natsjwtserver

import (
	"encoding/base64"
	"fmt"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

// NatsAuthnTokenizer generates and validates NATS NKey or JWT tokens.
// This implements the IAuthnTokenizer interface
type NatsAuthnTokenizer struct {
	accountKey nkeys.KeyPair
	// use nkey or jwt for tokens
	useNKeyToken bool
}

// ValidateJWTToken checks if the given token belongs the clientID and is valid.
//   - verify if jwtToken is a valid token
//   - validate the token isn't expired
//   - verify the user's public key's nonce based signature
//     this can only be signed when the user has its private key
//   - verify the issuer is the signing/account key.
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (svc *NatsAuthnTokenizer) ValidateJWTToken(
	clientID string, pubKey string, jwtToken string, signedNonce string, nonce string) (err error) {

	// the jwt token is not in the JWT field. Workaround by storing it in the token field.
	juc, err := jwt.DecodeUserClaims(jwtToken)
	if err != nil {
		return fmt.Errorf("unable to decode jwt token:%w", err)
	}
	// validate the jwt user claims (not expired)
	vr := jwt.CreateValidationResults()
	juc.Validate(vr)
	if len(vr.Errors()) > 0 {
		return fmt.Errorf("jwt authn failed: %w", vr.Errors()[0])
	}
	// the subject contains the public user nkey
	userPub, err := nkeys.FromPublicKey(juc.Subject)
	if err != nil {
		return fmt.Errorf("user nkey not valid: %w", err)
	}

	// Verify the nonce based token signature
	if signedNonce != "" {
		sig, err := base64.RawURLEncoding.DecodeString(signedNonce)
		if err != nil {
			// Allow fallback to normal base64.
			sig, err = base64.StdEncoding.DecodeString(signedNonce)
			if err != nil {
				return fmt.Errorf("signature not valid base64: %w", err)
			}
		}
		// verify the signature of the public key using the nonce
		// this tells us the user public key is not forged
		if err = userPub.Verify([]byte(nonce), sig); err != nil {
			return fmt.Errorf("signature not verified")
		}
	}
	// verify issuer account matches
	accPub, _ := svc.accountKey.PublicKey()
	if juc.Issuer != accPub {
		return fmt.Errorf("JWT issuer is not known")
	}
	// clientID must match the user
	if juc.Name != clientID {
		return fmt.Errorf("clientID doesn't match user")
	}

	//if entry.PubKey != userPub {
	//	return fmt.Errorf("user %s public key mismatch", clientID)
	//}

	//acc, err := svc.ns.LookupAccount(juc.IssuerAccount)
	//if err != nil {
	//	return fmt.Errorf("JWT issuer is not known")
	//}
	//if acc.IsExpired() {
	//	return fmt.Errorf("Account JWT has expired")
	//}
	// no access to account revocation list
	//if acc.checkUserRevoked(juc.Subject, juc.IssuedAt) {
	//	return fmt.Errorf("User authentication revoked")
	//}

	//if !validateSrc(juc, c.host) {
	//	return fmt.Errorf("Bad src Ip %s", c.host)
	//	return false
	//}
	return nil
}

// ValidateToken checks if the given token belongs the clientID and is valid.
// When keys is used this returns success
// When nkeys is not used this validates the JWT token
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (svc *NatsAuthnTokenizer) ValidateToken(
	clientID string, pubKey string, oldToken string, signedNonce string, nonce string) (err error) {
	if svc.useNKeyToken {
		if oldToken == "" || pubKey != oldToken {
			return fmt.Errorf("invalid old token for client '%s'", clientID)
		}
		return nil
	}
	return svc.ValidateJWTToken(clientID, pubKey, oldToken, signedNonce, nonce)
}

// NewNatsAuthnTokenizer handles token generation and verification for the NATS messaging server
//
//	accountKey is used to sign JWT tokens
//	useNKeyTokens to not use JWT and return public key as token
func NewNatsAuthnTokenizer(accountKey nkeys.KeyPair, useNKeyTokens bool) *NatsAuthnTokenizer {

	tokenizer := &NatsAuthnTokenizer{
		accountKey:   accountKey,
		useNKeyToken: useNKeyTokens,
	}
	return tokenizer
}
