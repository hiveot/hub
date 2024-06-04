package natstransport_test

import (
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/mqtttransport"
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

	roles := []string{api.ClientRoleViewer}
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
	kp := s.CreateKeyPair()

	_ = certBundle
	_ = cfg
	clInfo := mqtttransport.ClientAuthInfo{
		ClientID:   user2ID,
		ClientType: authapi.ClientTypeUser,
		PubKey:     kp.ExportPublic(),
		Role:       authapi.ClientRoleViewer,
	}
	user2Token, err := s.CreateToken(clInfo)
	require.NoError(t, err)

	err = s.ValidateToken(user2ID, user2Token, "", "")
	assert.NoError(t, err)
}
