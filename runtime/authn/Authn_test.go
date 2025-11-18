package authn_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/hiveot/gocore/logging"
	"github.com/hiveot/hub/runtime/authn/config"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/stretchr/testify/assert"
)

var testDir = path.Join(os.TempDir(), "test-authn")
var authnConfig config.AuthnConfig
var defaultHash = config.PWHASH_ARGON2id

// This test file sets up the environment for testing authn admin and client services.

// launch the authn service and return the server side message handlers for using and managing it.
func startTestAuthnService(testHash string) (
	svc *service.AuthnService, stopFn func()) {

	_ = os.RemoveAll(testDir)
	_ = os.MkdirAll(testDir, 0700)

	// the password file to use
	passwordFile := path.Join(testDir, "test.passwd")

	authnConfig = config.NewAuthnConfig()
	authnConfig.Setup(testDir, testDir)
	authnConfig.PasswordFile = passwordFile
	authnConfig.AgentTokenValidityDays = 1
	authnConfig.Encryption = testHash

	svc, err := service.StartAuthnService(&authnConfig)
	if err != nil {
		panic("Error starting authn admin service:" + err.Error())
	}
	//ag, err := service.StartAuthnAgent(svc, nil)

	//return svc, ag.HandleMessage, func() {
	return svc, func() {
		svc.Stop()

		// let background tasks finish
		time.Sleep(time.Millisecond * 100)
	}
}

// TestMain creates a test environment
// Used for all test cases in this package
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

// Start the authn service and list clients
func TestStartStop(t *testing.T) {
	// this creates the admin user key
	svc, stopFn := startTestAuthnService(defaultHash)
	assert.NotNil(t, svc.AdminSvc)
	assert.NotNil(t, svc.UserSvc)
	assert.NotNil(t, svc.AuthnStore)
	assert.NotNil(t, svc.SessionAuth)
	defer stopFn()
}
