package natsnkeyserver_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestPermissions(t *testing.T) {
	logrus.Infof("---TestPermissions start---")
	defer logrus.Infof("---TestPermissions end---")

	// setup
	clientURL, s, certBundle, cfg, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	_ = certBundle
	_ = clientURL
	_ = cfg

	roles := []string{auth.ClientRoleViewer}
	s.SetServicePermissions("myservice", "capability", roles)
}

func TestToken(t *testing.T) {
	logrus.Infof("---TestToken start---")
	defer logrus.Infof("---TestToken end---")

	// setup
	clientURL, s, certBundle, cfg, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	_ = certBundle
	_ = clientURL
	_ = cfg

	token, err := s.CreateToken(testenv.TestUser2ID, auth.ClientTypeUser, testenv.TestUser2Pub, 0)
	require.NoError(t, err)

	err = s.ValidateToken(testenv.TestUser2ID, testenv.TestUser2Pub, token, "", "")
	assert.NoError(t, err)
}
