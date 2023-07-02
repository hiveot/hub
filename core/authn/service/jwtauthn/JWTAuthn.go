package jwtauthn

import (
	"crypto/ecdsa"
	"fmt"
	"github.com/hiveot/hub/core/authn"
	"time"

	"github.com/golang-jwt/jwt"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/lib/certs"
)

// JWTAuthn provides creation and verification of JWT access and refresh tokens.
// Intended for use in user authentication.
type JWTAuthn struct {
	// private key used to sign the token
	signingKey      *ecdsa.PrivateKey
	verificationKey *ecdsa.PublicKey

	accessTokenValidity  uint
	refreshTokenValidity uint

	// invalidated tokens with their expiry unix time for auto cleaning
	invalidatedTokens map[string]int64
}

// CreateToken creates JWT session token for the given user and client type (audience)
func (jwtauthn *JWTAuthn) CreateToken(
	userID string, clientType string, validitySec uint) (token string, err error) {

	expTime := time.Now().Add(time.Second * time.Duration(validitySec))
	if userID == "" {
		err = fmt.Errorf("CreateJWTTokens: Missing userID")
		return "", err
	}

	// Create the JWT claims, which includes the userID, clientType, and expiry time
	claims := &jwt.StandardClaims{
		Issuer:   "hub",
		Audience: clientType,
		Subject:  userID,
		Id:       uuid.New().String(),
		// In JWT, the expiry time is expressed as unix milliseconds
		ExpiresAt: expTime.Unix(),
		IssuedAt:  time.Now().Unix(),
		NotBefore: time.Now().Unix(),
	}
	jwtToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	token, _ = jwtToken.SignedString(jwtauthn.signingKey)
	return token, nil
}

// CreateTokens creates JWT access and refresh tokens for the given user
func (jwtauthn *JWTAuthn) CreateTokens(userID string) (accessToken, refreshToken string, err error) {

	logrus.Debugf("create tokens for user '%s'. Access token valid for %d seconds, refresh for %d seconds",
		userID, jwtauthn.accessTokenValidity,
		jwtauthn.refreshTokenValidity)

	accessToken, _ = jwtauthn.CreateToken(userID, "", jwtauthn.accessTokenValidity)
	refreshToken, err = jwtauthn.CreateToken(userID, "", jwtauthn.accessTokenValidity)

	return accessToken, refreshToken, err
}

// InvalidateToken invalidates a refresh token. Intended for use by logout.
// If the token is invalid or expired it is ignored.
func (jwtauthn *JWTAuthn) InvalidateToken(userID string, refreshToken string) {
	// only add the token if it is still valid
	_, claims, err := jwtauthn.ValidateToken(userID, refreshToken)
	if err == nil {
		jwtauthn.invalidatedTokens[refreshToken] = claims.ExpiresAt
	}
}

// RefreshTokens refreshes a access/refresh token pair for a given user after validation of the existing refresh token.
//
//	userID is the ID for whom to issue the refresh token. Must match the existing refresh token
//	refreshToken must be a valid refresh token
//
// This returns a short lived auth token and medium lived refresh token
func (jwtauthn *JWTAuthn) RefreshTokens(userID string, refreshToken string) (
	newAccessToken, newRefreshToken string, err error) {

	// validate the provided token
	_, _, err = jwtauthn.ValidateToken(userID, refreshToken)

	if err != nil {
		// given refresh token is invalid. Authorization refused
		err = fmt.Errorf("given refresh token for user '%s' is invalid: %s", userID, err)
		return "", "", err
	}

	// create new token
	return jwtauthn.CreateTokens(userID)
}

// ValidateToken decodes and validates the token and return its claims.
// The token must be a valid JWT token, and:
//   - match against the verification public key
//   - not be expired
//   - issued to the given userID
//   - issued by this service
//   - not be invalidated
//
// If the token is invalid then claims will be empty and an error is returned
// If the token is valid but has an incorrect signature, the token and claims will be returned with an error
func (jwtauthn *JWTAuthn) ValidateToken(userID, tokenString string) (
	jwtToken *jwt.Token, claims *jwt.StandardClaims, err error) {

	claims = &jwt.StandardClaims{}
	jwtToken, err = jwt.ParseWithClaims(tokenString, claims,
		func(token *jwt.Token) (interface{}, error) {
			return jwtauthn.verificationKey, nil
		})
	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return jwtToken, claims, fmt.Errorf("invalid JWT token. Err=%s", err)
	}
	// StandardClaims.Valid ignores missing 0 dates so check ourselves
	//err = jwtToken.Claims.Valid()
	now := time.Now().Unix()
	if err == nil && !claims.VerifyExpiresAt(now, true) {
		delta := time.Unix(now, 0).Sub(time.Unix(claims.ExpiresAt, 0))
		err = fmt.Errorf("token of user '%s' is expired by %v", delta, userID)
	}
	if err == nil && !claims.VerifyIssuedAt(now, true) {
		err = fmt.Errorf("token of user '%s' used before issued", userID)
	}
	if err == nil && !claims.VerifyNotBefore(now, true) {
		err = fmt.Errorf("token of user '%s' is not valid yet", userID)
	}

	if err != nil {
		return jwtToken, claims, err
	}
	claims = jwtToken.Claims.(*jwt.StandardClaims)

	// check if token is for the given user
	if claims.Subject != userID {
		return jwtToken, claims, fmt.Errorf("token is issued to '%s', not to '%s'", claims.Subject, userID)
	}
	// check if token isn't revoked
	if _, revoked := jwtauthn.invalidatedTokens[tokenString]; revoked {
		return jwtToken, claims, fmt.Errorf("use of revoked token for user %s", userID)
	}
	return jwtToken, claims, nil
}

// NewJWTAuthn creates a new instance of the JWT token authenticator
// A private key for signing and verification can be provided in case existing tokens must remain valid.
// To invalidate existing tokens, provide nil as the signing key or a new key.
//
//	signingKey is the private key used for signing. Use nil to have one auto-generated.
//	accessTokenValidity in seconds. Use 0 for default.
//	refreshTokenValidity in seconds. Use 0 for default.
func NewJWTAuthn(signingKey *ecdsa.PrivateKey, accessTokenValidity uint, refreshTokenValidity uint) *JWTAuthn {
	if signingKey == nil {
		signingKey = certs.CreateECDSAKeys()
	}
	if accessTokenValidity == 0 {
		accessTokenValidity = uint(authn.DefaultAccessTokenValiditySec)
	}
	if refreshTokenValidity == 0 {
		refreshTokenValidity = uint(authn.DefaultRefreshTokenValiditySec)
	}

	jwtauthn := &JWTAuthn{
		signingKey:           signingKey,
		verificationKey:      &signingKey.PublicKey,
		accessTokenValidity:  accessTokenValidity,
		refreshTokenValidity: refreshTokenValidity,
		invalidatedTokens:    make(map[string]int64),
	}
	return jwtauthn
}
