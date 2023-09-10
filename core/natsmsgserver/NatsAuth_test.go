package natsmsgserver_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPermissions(t *testing.T) {
	t.Log("---TestPermissions start---")
	defer t.Log("---TestPermissions end---")

	// setup
	clientURL, s, certBundle, cfg, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	_ = certBundle
	_ = clientURL
	_ = cfg

	roles := []string{auth.ClientRoleViewer}
	s.SetServicePermissions("myservice", "capability", roles)
}

func TestToken(t *testing.T) {
	t.Log("---TestToken start---")
	defer t.Log("---TestToken end---")

	// setup
	clientURL, s, certBundle, cfg, err := testenv.StartNatsTestServer(withCallout)
	require.NoError(t, err)
	defer s.Stop()
	err = s.ApplyAuth(testenv.TestClients)
	require.NoError(t, err)

	_ = certBundle
	_ = clientURL
	_ = cfg
	user2Token, err := s.CreateToken(testenv.TestUser2ID, testenv.TestUser2Pub)
	require.NoError(t, err)

	err = s.ValidateToken(testenv.TestUser2ID, testenv.TestUser2Pub, user2Token, "", "")
	assert.NoError(t, err)
}
