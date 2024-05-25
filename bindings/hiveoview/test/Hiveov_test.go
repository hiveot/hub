package test

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/service"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

const core = "mqtt"
const serviceID = "hiveoview"

var testFolder = path.Join(os.TempDir(), "test-hiveoview")

// the following are set by the testmain
var testServer *testenv.TestServer
var serverURL string

func TestMain(m *testing.M) {
	var err error
	logging.SetLogging("info", "")
	// clean start
	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)

	testServer, err = testenv.StartTestServer(core, true)
	serverURL, _, _ = testServer.MsgServer.GetServerURLs()
	if err != nil {
		panic(err)
	}

	res := m.Run()
	testServer.Stop()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop ---")

	svc := service.NewHiveovService(9999, true, nil, "")
	hc1, err := testServer.AddConnectUser(
		serviceID, authapi.ClientTypeService, authapi.ClientRoleService)
	require.NoError(t, err)
	svc.Start(hc1)
	time.Sleep(time.Second * 3)
	svc.Stop()
}
