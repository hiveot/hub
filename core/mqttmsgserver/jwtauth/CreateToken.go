package jwtauth

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"time"
)

// CreateToken creates a JWT authtoken for a client.
func CreateToken(authInfo msgserver.ClientAuthInfo, signingKey *ecdsa.PrivateKey) (token string, err error) {

	if authInfo.ClientID == "" || authInfo.ClientType == "" {
		err = fmt.Errorf("CreateToken: Missing client ID or type")
		return "", err
	} else if authInfo.PubKey == "" {
		err = fmt.Errorf("CreateToken: client has no public key")
		return "", err
	} else if authInfo.Role == "" {
		err = fmt.Errorf("CreateToken: client has no role")
		return "", err
	}

	// see also: https://golang-jwt.github.io/jwt/usage/create/
	// TBD: use validity period from profile
	// default validity period depends on client type (why?)
	validity := auth.DefaultUserTokenValidityDays
	if authInfo.ClientType == auth.ClientTypeDevice {
		validity = auth.DefaultDeviceTokenValidityDays
	} else if authInfo.ClientType == auth.ClientTypeService {
		validity = auth.DefaultServiceTokenValidityDays
	}
	expiryTime := time.Now().Add(time.Duration(validity) * time.Hour * 24)

	signingKeyPub, _ := x509.MarshalPKIXPublicKey(&signingKey.PublicKey)
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)
	// Create the JWT claims, which includes the username, clientType and expiry time
	claims := jwt.MapClaims{
		//"alg": "ES256", // jwt.SigningMethodES256,
		"typ": "JWT",
		"aud": authInfo.ClientType, //
		"sub": authInfo.PubKey,     // public key of client (same as nats)
		"iss": signingKeyPubStr,
		"exp": expiryTime.Unix(), // expiry time. Seconds since epoch
		"iat": time.Now().Unix(), // issued at. Seconds since epoch

		// custom claim fields
		"clientID":   authInfo.ClientID,
		"clientType": authInfo.ClientType,
		"role":       authInfo.Role,
	}

	// Declare the token with the algorithm used for signing, and the claims
	claimsToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	authToken, err := claimsToken.SignedString(signingKey)
	if err != nil {
		return "", err
	}

	return authToken, nil
}
