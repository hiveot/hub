package jwtauthn_test

import (
	"github.com/hiveot/hub/core/authn/service/jwtauthn"
	"testing"

	"github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/hiveot/hub/lib/certsclient"
)

// JWT token creation and verification test cases
func TestCreateValidateJWTToken(t *testing.T) {
	user1 := "user1"

	// pub/private key for signing tokens
	issuer := jwtauthn.NewJWTAuthn(nil, 0, 0)

	accessToken, refreshToken, err := issuer.CreateTokens(user1)
	require.NoError(t, err)
	assert.NotEmpty(t, accessToken)
	assert.NotEmpty(t, refreshToken)

	// validate both access and refresh tokens
	token, claims, err := issuer.ValidateToken(user1, accessToken)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, claims)
	assert.NoError(t, err)

	token, claims, err = issuer.ValidateToken(user1, refreshToken)
	assert.NotEmpty(t, token)
	assert.NotEmpty(t, claims)
	assert.NoError(t, err)
}
func TestJWTMissingUserID(t *testing.T) {
	// issue the tokens
	issuer := jwtauthn.NewJWTAuthn(nil, 60, 60)
	at1, rt1, err := issuer.CreateTokens("")
	assert.Error(t, err)
	assert.Empty(t, at1)
	assert.Empty(t, rt1)
}

func TestJWTIncorrectVerificationKey(t *testing.T) {
	user1 := "user1"

	// issue the tokens
	privKey1 := certsclient.CreateECDSAKeys()
	issuer := jwtauthn.NewJWTAuthn(privKey1, 60, 60)
	at1, rt1, err := issuer.CreateTokens(user1)
	assert.NoError(t, err)

	// verification should succeed with own key
	decodedToken, claims, err := issuer.ValidateToken(user1, at1)
	assert.NoError(t, err)
	assert.NotEmpty(t, decodedToken)
	assert.NotNil(t, claims)

	// verification should fail using someone else's user id
	_, _, err = issuer.ValidateToken("someonelese", rt1)
	assert.Error(t, err)

	// verification should fail using someone else's key
	privKey2 := certsclient.CreateECDSAKeys()
	issuer2 := jwtauthn.NewJWTAuthn(privKey2, 60, 60)
	decodedToken2, claims2, err := issuer2.ValidateToken(user1, at1)
	assert.Error(t, err)
	assert.NotEmpty(t, decodedToken2)
	assert.NotNil(t, claims2)
}

func TestRefresh(t *testing.T) {
	const user1 = "user1"

	// issue the tokens
	privKey := certsclient.CreateECDSAKeys()
	issuer := jwtauthn.NewJWTAuthn(privKey, 0, 0)
	at1, rt1, err := issuer.CreateTokens(user1)
	assert.NotEmpty(t, at1)
	assert.NotEmpty(t, rt1)
	assert.NoError(t, err)

	_, rt2, err2 := issuer.RefreshTokens(user1, rt1)
	assert.NoError(t, err2)
	_, _, err2 = issuer.ValidateToken(user1, at1)
	assert.NoError(t, err2)
	_, _, err2 = issuer.ValidateToken(user1, rt2)
	assert.NoError(t, err2)

	// token refresh with different user fails
	_, _, err2 = issuer.RefreshTokens("someoneelse", rt1)
	assert.Error(t, err2)
}

func TestExpired(t *testing.T) {
	const user1 = "user1"

	// issue the tokens
	issuer := jwtauthn.NewJWTAuthn(nil, -1, 0)
	at1, rt1, err := issuer.CreateTokens(user1)
	assert.NotEmpty(t, at1)
	assert.NotEmpty(t, rt1)
	assert.NoError(t, err)

	_, _, err = issuer.RefreshTokens(user1, rt1)
	assert.NoError(t, err)
	// access token is expired
	_, _, err = issuer.ValidateToken(user1, at1)
	assert.Error(t, err)
}

func TestValidateWrongToken(t *testing.T) {
	const user1 = "user1"

	// issue the tokens
	privKey := certsclient.CreateECDSAKeys()
	issuer := jwtauthn.NewJWTAuthn(privKey, 60, 60)
	at1, rt1, err := issuer.CreateTokens(user1)
	assert.NotEmpty(t, at1)
	assert.NotEmpty(t, rt1)
	assert.NoError(t, err)

	// token validation with different issuer fails
	privKey2 := certsclient.CreateECDSAKeys()
	issuer2 := jwtauthn.NewJWTAuthn(privKey2, 60, 60)
	_, _, err2 := issuer2.ValidateToken(user1, rt1)
	assert.Error(t, err2)

	jwtAccessToken := jwt.New(jwt.SigningMethodES256)
	badToken, err := jwtAccessToken.SignedString(privKey)
	assert.NoError(t, err)
	_, _, err = issuer.ValidateToken(user1, badToken)
	assert.Error(t, err)
}

func TestInvalidateToken(t *testing.T) {
	const user1 = "user1"

	// issue the tokens
	privKey := certsclient.CreateECDSAKeys()
	issuer := jwtauthn.NewJWTAuthn(privKey, 60, 60)
	_, rt1, err := issuer.CreateTokens(user1)
	assert.NoError(t, err)
	//
	issuer.InvalidateToken(user1, rt1)
	_, rt2, err2 := issuer.RefreshTokens(user1, rt1)
	assert.Error(t, err2)
	assert.Empty(t, rt2)
}
