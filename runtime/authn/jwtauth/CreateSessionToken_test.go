package jwtauth_test

import (
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/jwtauth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestCreateSessionToken(t *testing.T) {
	const clientID = "user1"
	const clientType = authn.ClientTypeUser
	sessionID := "session1"
	signingKey := keys.NewEcdsaKey()

	token1, err := jwtauth.CreateSessionToken(clientID, sessionID, signingKey, 100)
	require.NoError(t, err)
	assert.NotEmpty(t, token1)

	// decode it
	cid2, sid2, err := jwtauth.DecodeSessionToken(token1, signingKey.PublicKey(), "", "")
	require.NoError(t, err)
	require.Equal(t, clientID, cid2)
	require.Equal(t, sessionID, sid2)

}
