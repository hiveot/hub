package jwtauth_test

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/msgserver"
	jwtauth2 "github.com/hiveot/hub/core/msgserver/mqttmsgserver/jwtauth"
	"github.com/hiveot/hub/lib/certs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateToken(t *testing.T) {
	const clientID = "user1"
	const clientType = authapi.ClientTypeUser
	signingKey, _ := certs.CreateECDSAKeys()
	_, pubKey := certs.CreateECDSAKeys()

	authInfo := msgserver.ClientAuthInfo{
		ClientID:   clientID,
		ClientType: clientType,
		PubKey:     pubKey,
		Role:       authapi.ClientRoleAdmin,
	}
	token1, err := jwtauth2.CreateToken(authInfo, signingKey)
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	// next, validate the created token
	claims, err := jwtauth2.ValidateToken(clientID, token1, signingKey, "", "")
	require.NoError(t, err)
	assert.Equal(t, clientID, claims.ClientID)
}
