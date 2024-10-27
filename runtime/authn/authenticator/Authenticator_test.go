package authenticator_test

import (
	"crypto/ed25519"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn/authenticator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var authnStore api.IAuthnStore

func NewAuthenticator() api.IAuthenticator {
	//signingKey := keys.NewEcdsaKey()
	//svc := authenticator.NewJWTAuthenticator(authnStore, signingKey)
	signingKey := keys.NewEd25519Key().PrivateKey().(ed25519.PrivateKey)
	svc := authenticator.NewPasetoAuthenticator(authnStore, signingKey)
	return svc
}

func TestCreateSessionToken(t *testing.T) {
	const clientID = "user1"
	//const clientType = authn.ClientTypeConsumer
	sessionID := "session1"

	svc := NewAuthenticator()

	token1 := svc.CreateSessionToken(clientID, sessionID, 100)
	assert.NotEmpty(t, token1)

	// decode it
	clientID2, sid2, err := svc.DecodeSessionToken(token1, "", "")
	require.NoError(t, err)
	require.Equal(t, clientID, clientID2)
	require.Equal(t, sessionID, sid2)

	// validate the new token. Without a session this fails
	clientID3, sid3, err := svc.ValidateToken(token1)
	require.Error(t, err)
	require.Equal(t, clientID, clientID3)
	require.Equal(t, sessionID, sid3)

	// create a persistent session token (use clientID as sessionID)
	token2 := svc.CreateSessionToken(clientID, clientID, 100)
	clientID4, sid4, err := svc.ValidateToken(token2)
	require.NoError(t, err)
	require.Equal(t, clientID, clientID4)
	require.Equal(t, clientID, sid4)

}

func TestBadTokens(t *testing.T) {
	const clientID = "user1"
	//const clientType = authn.ClientTypeConsumer
	sessionID := "session1"

	svc := NewAuthenticator()

	token1 := svc.CreateSessionToken(clientID, sessionID, 100)
	assert.NotEmpty(t, token1)

	// try to refresh as a different client

	// refresh
	badToken := token1 + "-bad"
	_, _, err := svc.ValidateToken(badToken)
	require.Error(t, err)

	// expired
	token2 := svc.CreateSessionToken(clientID, sessionID, -100)
	clientID2, sid2, err := svc.ValidateToken(token2)
	require.Error(t, err)
	assert.Empty(t, clientID2)
	assert.Empty(t, sid2)

}
