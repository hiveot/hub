package old

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"log/slog"
	"net/http"
)

// JwtClaims this is temporary while figuring things out
type JwtClaims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTAuthenticator verifies issued JWT access token using the provided public key.
// See JWTIssuer for test cases of the authenticator.
// The application must use .AuthenticateRequest() to authenticate the incoming request using the
// access token.
type JWTAuthenticator struct {
	// Service certificate whose public key is used for token verification
	publicKey *ecdsa.PublicKey
}

// AuthenticateRequest validates the access token
// The access token is provided in the request header using the Bearer schema:
//
//	Authorization: Bearer <token>
//
// Returns the authenticated user and true if there is a match, of false if authentication failed
func (jauth *JWTAuthenticator) AuthenticateRequest(resp http.ResponseWriter, req *http.Request) (userID string, match bool) {

	accessTokenString, err := GetBearerToken(req)
	if err != nil {
		// this just means JWT is not used
		slog.Debug("JWTAuthenticator: No bearer token in request",
			"method", req.Method, "uri", req.RequestURI, "remoteAddr", req.RemoteAddr)
		return "", false
	}
	jwtToken, claims, err := jauth.DecodeToken(accessTokenString)
	_ = jwtToken
	_ = claims
	if err != nil {
		// token needs a refresh
		slog.Info("JWTAuthenticator: Invalid access token in request",
			"method", req.Method, "uri", req.RequestURI, "remoteAddr", req.RemoteAddr, err)
		return "", false
	}
	// TODO: verify claims: iat, iss, aud

	// hoora its valid
	slog.Debug("JWTAuthenticator. PubAction by authenticated with valid JWT token")
	return claims.Username, true
}

// DecodeToken and return its claims
//
// If the token is invalid then claims will be empty and an error is returned
// If the token is valid but has an incorrect signature, the token and claims will be returned with an error
func (jauth *JWTAuthenticator) DecodeToken(tokenString string) (
	jwtToken *jwt.Token, claims *JwtClaims, err error) {

	claims = &JwtClaims{}
	jwtToken, err = jwt.ParseWithClaims(tokenString, claims,
		func(token *jwt.Token) (interface{}, error) {
			return jauth.publicKey, nil
		})
	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return jwtToken, claims, fmt.Errorf("invalid JWT token. Err=%s", err)
	}
	claims = jwtToken.Claims.(*JwtClaims)

	return jwtToken, claims, nil
}

// NewJWTAuthenticator creates a new JWT authenticator
// publicKey is the public key for verifying the private key signature
func NewJWTAuthenticator(publicKey *ecdsa.PublicKey) *JWTAuthenticator {
	//publicKeyDer, _ := x509.MarshalPKIXPublicKey(pubKey)

	ja := &JWTAuthenticator{publicKey: publicKey}
	return ja
}
