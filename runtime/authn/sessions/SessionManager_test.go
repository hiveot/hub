package sessions_test

import (
	"github.com/hiveot/hivehub/runtime/authn/sessions"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAddRemoveSession(t *testing.T) {
	const clientID = "client1"
	const session1ID = "sess1"

	sm := sessions.NewSessionmanager()
	sm.NewSession(clientID, session1ID)

	// session must exist
	s1, found := sm.GetSessionBySessionID(session1ID)
	require.True(t, found)
	require.Equal(t, clientID, s1.ClientID)
	require.Equal(t, session1ID, s1.SessionID)

	s1, found = sm.GetSessionByClientID(clientID)
	require.True(t, found)
	require.Equal(t, clientID, s1.ClientID)
	require.Equal(t, session1ID, s1.SessionID)

	// close the session
	sm.Remove(session1ID)

	// session no longer exists
	s1, found = sm.GetSessionBySessionID(session1ID)
	require.False(t, found)
	require.Empty(t, s1)
	s1, found = sm.GetSessionByClientID(clientID)
	require.False(t, found)
	require.Empty(t, s1)
}

func TestReplaceSession(t *testing.T) {
	const clientID = "client1"
	const session1ID = "sess1"
	const session2ID = "sess2"

	sm := sessions.NewSessionmanager()
	sm.NewSession(clientID, session1ID)
	sm.NewSession(clientID, session2ID)

	// session1 should not exist
	s1, found := sm.GetSessionBySessionID(session1ID)
	require.False(t, found)
	require.Empty(t, s1)

	// session2 should exist

	// client should be linked to session 2
	s2, found := sm.GetSessionByClientID(clientID)
	require.True(t, found)
	require.Equal(t, clientID, s2.ClientID)
	require.Equal(t, session2ID, s2.SessionID)
}

func TestRemoveAll(t *testing.T) {
	const client1ID = "client1"
	const session1ID = "sess1"
	const client2ID = "client2"
	const session2ID = "sess2"

	sm := sessions.NewSessionmanager()
	sm.NewSession(client1ID, session1ID)
	sm.NewSession(client2ID, session2ID)

	// sessions must exist
	_, found := sm.GetSessionBySessionID(session1ID)
	require.True(t, found)
	_, found = sm.GetSessionBySessionID(session2ID)
	require.True(t, found)

	// after removing all neither must exist
	sm.RemoveAll()
	_, found = sm.GetSessionBySessionID(session1ID)
	require.False(t, found)
	_, found = sm.GetSessionBySessionID(session2ID)
	require.False(t, found)
}

func TestSessionRenewal(t *testing.T) {
	const clientID = "client1"
	const sessionID = "sess1"
	sm := sessions.NewSessionmanager()
	sm.NewSession(clientID, sessionID)
	s1, found := sm.GetSessionBySessionID(sessionID)

	// renew session and must have later expiry
	time.Sleep(time.Millisecond * 100)
	sm.NewSession(clientID, sessionID)
	require.True(t, found)
	require.Equal(t, clientID, s1.ClientID)

	// renew session and must have later expiry
	s2, found := sm.GetSessionBySessionID(sessionID)
	require.Greater(t, s2.Expiry, s1.Expiry)
}

func TestSessionFail(t *testing.T) {
	const clientID = "client1"
	const remoteAddr = "remoteAddr"
	const sessionID = "sess1"

	sm := sessions.NewSessionmanager()

	// add bad clientID
	sm.NewSession("", sessionID)
	s1, found := sm.GetSessionByClientID("")
	require.False(t, found)
	require.Empty(t, s1)
	s1, found = sm.GetSessionBySessionID(sessionID)
	require.False(t, found)
	require.Empty(t, s1)

	// add bad sessionID
	sm.NewSession(clientID, "")
	s1, found = sm.GetSessionByClientID(clientID)
	require.False(t, found)
	require.Empty(t, s1)
	s1, found = sm.GetSessionBySessionID(sessionID)
	require.False(t, found)
	require.Empty(t, s1)
}
