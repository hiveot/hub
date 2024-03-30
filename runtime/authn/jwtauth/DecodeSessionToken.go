package jwtauth

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
)

// DecodeSessionToken verifies the given JWT token and returns its claims.
// optionally verify the signed nonce using the client's public key.
// This returns the auth info stored in the token.
//
// token is the jwt token string containing a session token
// pubKey is the public key of the keypair used to sign the token. ECDSA or EdDSA
// This returns the client info reconstructed from the token or an error if invalid
func DecodeSessionToken(token string, pubKey interface{}, signedNonce string, nonce string) (
	clientID string, sessionID string, err error) {

	signingKeyPub, _ := x509.MarshalPKIXPublicKey(pubKey)
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	claims := jwt.MapClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, &claims,
		func(token *jwt.Token) (interface{}, error) {
			return pubKey, nil
		}, jwt.WithValidMethods([]string{
			jwt.SigningMethodES256.Name,
			jwt.SigningMethodES384.Name,
			jwt.SigningMethodES512.Name,
			"EdDSA",
		}),
		jwt.WithIssuer(signingKeyPubStr), // url encoded string
	)
	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return "", "",
			fmt.Errorf("ValidateToken: invalid JWT token or signing key: %s", err)
	}
	cid := claims["clientID"]
	if cid != nil {
		clientID = cid.(string)
	}
	sid := claims["sessionID"]
	if sid != nil {
		sessionID = sid.(string)
	}
	return clientID, sessionID, nil
}
