package jwtauth

import (
	"crypto/x509"
	"encoding/base64"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hub/lib/keys"
	"time"
)

// CreateSessionToken creates a signed JWT session token for a client.
// The token is constructed with MapClaims containing "ID" as session ID and
// "clientID" identifying the connectied client.
// The token is signed with the given signing key-pair and valid for the given duration.
func CreateSessionToken(clientID, sessionID string, signingKey keys.IHiveKey, validitySec int) (token string, err error) {

	validity := time.Second * time.Duration(validitySec)
	expiryTime := time.Now().Add(validity)

	signingKeyPub, _ := x509.MarshalPKIXPublicKey(signingKey.PublicKey())
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)

	// Create the JWT claims, which includes the username, clientType and expiry time
	claims := jwt.MapClaims{
		//"alg": "ES256", // jwt.SigningMethodES256,
		"typ": "JWT",
		//"aud": authInfo.ClientID, // recipient of the jwt
		"sub": clientID,          // subject of the jwt, eg the client
		"iss": signingKeyPubStr,  // issuer of the jwt (public key)
		"exp": expiryTime.Unix(), // expiry time. Seconds since epoch
		"iat": time.Now().Unix(), // issued at. Seconds since epoch

		// custom claim fields
		"clientID":  clientID,
		"sessionID": sessionID,
	}

	// Declare the token with the algorithm used for signing, and the claims
	claimsToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	authToken, err := claimsToken.SignedString(signingKey.PrivateKey())

	if err != nil {
		return "", err
	}

	return authToken, nil
}
