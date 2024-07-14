package authenticator_test

import (
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authenticator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var authnStore api.IAuthnStore

func TestCreateSessionToken(t *testing.T) {
	const clientID = "user1"
	//const clientType = authn.ClientTypeConsumer
	sessionID := "session1"

	signingKey := keys.NewEcdsaKey()
	svc := authenticator.NewJWTAuthenticator(authnStore, signingKey)

	token1 := svc.CreateSessionToken(clientID, sessionID, 100)
	assert.NotEmpty(t, token1)

	// decode it
	cid2, sid2, err := svc.DecodeSessionToken(token1, "", "")
	require.NoError(t, err)
	require.Equal(t, clientID, cid2)
	require.Equal(t, sessionID, sid2)

	// validate the new token
	cid3, sid3, err := svc.ValidateToken(token1)
	require.NoError(t, err)
	require.Equal(t, clientID, cid3)
	require.Equal(t, sessionID, sid3)
}

func TestBadTokens(t *testing.T) {
	const clientID = "user1"
	//const clientType = authn.ClientTypeConsumer
	sessionID := "session1"

	signingKey := keys.NewEcdsaKey()
	svc := authenticator.NewJWTAuthenticator(authnStore, signingKey)

	token1 := svc.CreateSessionToken(clientID, sessionID, 100)
	assert.NotEmpty(t, token1)

	// try to refresh as a different client

	// refresh
	badToken := token1 + "-bad"
	_, _, err := svc.ValidateToken(badToken)
	require.Error(t, err)

	// expired
	token2 := svc.CreateSessionToken(clientID, sessionID, -100)
	cid2, sid2, err := svc.ValidateToken(token2)
	require.Error(t, err)
	assert.Equal(t, clientID, cid2)
	assert.Equal(t, sessionID, sid2)

}
