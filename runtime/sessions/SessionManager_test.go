package sessions_test

import (
	"fmt"
	"github.com/hiveot/hub/runtime/sessions"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddRemoveSession(t *testing.T) {
	const clientID = "client1"
	const remoteAddr = "remote1"
	const session1ID = "sess1"
	const session2ID = "sess2"

	sm := sessions.NewSessionmanager()
	s, err := sm.AddSession(clientID, remoteAddr, session1ID)
	require.NoError(t, err)
	require.NotEmpty(t, s)
	s, err = sm.AddSession(clientID, remoteAddr, session2ID)
	require.NoError(t, err)
	require.NotEmpty(t, s)
	require.Equal(t, clientID, s.GetClientID())
	require.Equal(t, session2ID, s.GetSessionID())

	// session must exist
	s, err = sm.GetSession(session1ID)
	require.NoError(t, err)
	require.NotEmpty(t, s)

	sList, err := sm.GetSessionsByClientID(clientID)
	require.NoError(t, err)
	require.Equal(t, 2, len(sList))

	// close the session
	err = sm.Remove(session1ID)
	require.NoError(t, err)

	// session no longer exists
	s, err = sm.GetSession(session1ID)
	require.Error(t, err)
	require.Empty(t, s)

	// close  session 2
	err = sm.Remove(session2ID)
	require.NoError(t, err)

	sList, err = sm.GetSessionsByClientID(clientID)
	require.Error(t, err)

	// close the session again
	err = sm.Remove(session1ID)
	require.Error(t, err)

	// close all
	sm.RemoveAll()
}

func TestSessionFail(t *testing.T) {
	const clientID = "client1"
	const remoteAddr = "remoteAddr"
	const sessionID = "sess1"

	sm := sessions.NewSessionmanager()

	// add bad remote params
	s, err := sm.AddSession("", remoteAddr, sessionID)
	require.Error(t, err)
	require.Empty(t, s)
	s, err = sm.AddSession(clientID, "", sessionID)
	require.Error(t, err)
	require.Empty(t, s)
	s, err = sm.AddSession(clientID, remoteAddr, "")
	require.Error(t, err)
	require.Empty(t, s)

	// session must not exist
	s, err = sm.GetSession(sessionID)
	require.Error(t, err)
	require.Empty(t, s)
}

func TestTooManySessions(t *testing.T) {
	const clientID = "client1"
	const remoteAddr = "remoteAddr"
	var count = 1000
	var i = 0
	var err error

	sm := sessions.NewSessionmanager()

	for i = range count {
		// add bad remote params
		sid := fmt.Sprintf("session-%d", i)
		_, err = sm.AddSession(clientID, remoteAddr, sid)
		if err != nil {
			break
		}
	}
	sList, _ := sm.GetSessionsByClientID(clientID)
	assert.Error(t, err)
	assert.GreaterOrEqual(t, len(sList), 10)
}
