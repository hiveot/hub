package jwtauth

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/keys"
)

// ValidateToken verifies the given JWT token and returns its claims.
// optionally verify the signed nonce using the client's public key.
// This returns the auth info stored in the token.
func ValidateToken(clientID string, token string, signingKey keys.IHiveKey, signedNonce string, nonce string) (
	authInfo msgserver.ClientAuthInfo, err error) {

	signingKeyPub, _ := x509.MarshalPKIXPublicKey(signingKey.PublicKey())
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	claims := jwt.MapClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, &claims,
		func(token *jwt.Token) (interface{}, error) {
			return signingKey.PublicKey(), nil
		}, jwt.WithValidMethods([]string{
			jwt.SigningMethodES256.Name,
			jwt.SigningMethodES384.Name,
			jwt.SigningMethodES512.Name,
			"EdDSA",
		}),
		jwt.WithIssuer(signingKeyPubStr), // url encoded string
	)
	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return authInfo, fmt.Errorf("ValidateToken: client '%s' invalid JWT token or signing key: %s", clientID, err)
	}

	tokenPubKey, _ := claims.GetSubject()
	authInfo.PubKey = tokenPubKey

	jwtClientType, _ := claims["clientType"]
	authInfo.ClientType = jwtClientType.(string)
	if authInfo.ClientType == "" {
		return authInfo, fmt.Errorf("ValidateToken: client '%s' token has no client type", clientID)
	}

	authInfo.ClientID = clientID
	jwtClientID, _ := claims["clientID"]
	if jwtClientID != clientID {
		// while this doesn't provide much extra security it might help
		// prevent bugs. Potentially also useful as second factor auth check if
		// clientID is obtained through a different means.
		return authInfo, fmt.Errorf("ValidateToken: client '%s' token belongs to different clientID (%s)", clientID, jwtClientID)
	}

	jwtRole, _ := claims["role"]
	authInfo.Role = jwtRole.(string)
	if authInfo.Role == "" {
		return authInfo, fmt.Errorf("ValidateToken: client '%s' token has no role", clientID)
	}

	return authInfo, nil
}
