package authenticator_test

import (
	"crypto/ed25519"
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/hivekit/go/lib/keys"
	"github.com/hiveot/hivekit/go/lib/messaging"
	authn "github.com/hiveot/hub/runtime/authn/api"
	"github.com/hiveot/hub/runtime/authn/authenticator"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/config"
	"github.com/hiveot/hub/runtime/authn/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var authnStore authnstore.IAuthnStore
var testDir = path.Join(os.TempDir(), "test-authn")
var defaultHash = config.PWHASH_ARGON2id

func NewAuthenticator() (messaging.IAuthenticator, *sessions.SessionManager) {
	passwordFile := path.Join(testDir, "test.passwd")
	authnStore = authnstore.NewAuthnFileStore(passwordFile, defaultHash)
	//signingKey := keys.NewEcdsaKey()
	//svc := authenticator.NewJWTAuthenticator(authnStore, signingKey)
	signingKey := keys.NewEd25519Key().PrivateKey().(ed25519.PrivateKey)
	sm := sessions.NewSessionmanager()
	svc := authenticator.NewPasetoAuthenticator(authnStore, signingKey, sm)
	svc.SetAuthServerURI("/fake/server/endpoint")
	return svc, sm
}

func TestCreateSessionToken(t *testing.T) {
	const clientID = "user1"
	const pass1 = "pass1"
	//const clientType = authn.ClientTypeConsumer
	sessionID := "session1"

	svc, sm := NewAuthenticator()
	_ = authnStore.Add(clientID, authn.ClientProfile{
		ClientID:    clientID,
		ClientType:  authn.ClientTypeConsumer,
		Disabled:    false,
		DisplayName: "test",
	})
	err := authnStore.SetPassword(clientID, pass1)
	require.NoError(t, err)

	token1 := svc.CreateSessionToken(clientID, sessionID, time.Minute)
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

	_, err = svc.Login(clientID, pass1)

	// create a persistent session token (use clientID as sessionID)
	token2 := svc.CreateSessionToken(clientID, clientID, time.Minute)
	clientID4, sid4, err := svc.ValidateToken(token2)
	require.NoError(t, err)
	require.Equal(t, clientID, clientID4)
	require.Equal(t, clientID, sid4)

	// session info must exist
	sessInfo, found := sm.GetSessionByClientID(clientID)
	assert.True(t, found)
	assert.Equal(t, clientID4, sessInfo.ClientID)
	assert.NotEmpty(t, sessInfo.Created)
	assert.NotEmpty(t, sessInfo.Expiry)

}

func TestBadTokens(t *testing.T) {
	const clientID = "user1"
	//const clientType = authn.ClientTypeConsumer
	sessionID := "session1"

	svc, _ := NewAuthenticator()

	token1 := svc.CreateSessionToken(clientID, sessionID, time.Minute)
	assert.NotEmpty(t, token1)

	// try to refresh as a different client

	// refresh
	badToken := token1 + "-bad"
	_, _, err := svc.ValidateToken(badToken)
	require.Error(t, err)

	// expired
	token2 := svc.CreateSessionToken(clientID, sessionID, -1)
	clientID2, sid2, err := svc.ValidateToken(token2)
	require.Error(t, err)
	assert.Empty(t, clientID2)
	assert.Empty(t, sid2)

}
