package jwtauth_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/mqttmsgserver/jwtauth"
	"github.com/hiveot/hub/lib/certs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateToken(t *testing.T) {
	const clientID = "user1"
	const clientType = auth.ClientTypeUser
	signingKey, _ := certs.CreateECDSAKeys()
	_, pubKey := certs.CreateECDSAKeys()

	authInfo := msgserver.ClientAuthInfo{
		ClientID:   clientID,
		ClientType: clientType,
		PubKey:     pubKey,
		Role:       auth.ClientRoleAdmin,
	}
	token1, err := jwtauth.CreateToken(authInfo, signingKey)
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	// next, validate the created token
	claims, err := jwtauth.ValidateToken(clientID, token1, signingKey, "", "")
	require.NoError(t, err)
	assert.Equal(t, clientID, claims.ClientID)
}
