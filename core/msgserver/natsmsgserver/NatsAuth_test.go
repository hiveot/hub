package natsmsgserver_test

import (
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPermissions(t *testing.T) {
	t.Log("---TestPermissions start---")
	defer t.Log("---TestPermissions end---")

	// setup
	s, certBundle, cfg, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	_ = certBundle
	_ = cfg

	roles := []string{auth2.ClientRoleViewer}
	s.SetServicePermissions("myservice", "capability", roles)
}

func TestToken(t *testing.T) {
	t.Log("---TestToken start---")
	defer t.Log("---TestToken end---")

	// setup
	s, certBundle, cfg, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	//err = s.ApplyAuth(testenv.TestClients)
	require.NoError(t, err)
	user2ID := "user2"
	user2Key, user2Pub := s.CreateKP()
	_ = user2Key

	_ = certBundle
	_ = cfg
	clInfo := msgserver.ClientAuthInfo{
		ClientID:   user2ID,
		ClientType: auth2.ClientTypeUser,
		PubKey:     user2Pub,
		Role:       auth2.ClientRoleViewer,
	}
	user2Token, err := s.CreateToken(clInfo)
	require.NoError(t, err)

	err = s.ValidateToken(user2ID, user2Token, "", "")
	assert.NoError(t, err)
}