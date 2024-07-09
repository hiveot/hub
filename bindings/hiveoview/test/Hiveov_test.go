package test

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/service"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
	"time"
)

const serviceID = "hiveoview"

var testFolder = path.Join(os.TempDir(), "test-hiveoview")
var testCerts = certs.CreateTestCertBundle()

// the following are set by the testmain
var ts *testenv.TestServer

func TestMain(m *testing.M) {
	var err error
	logging.SetLogging("info", "")
	// clean start
	_ = os.RemoveAll(testFolder)
	_ = os.MkdirAll(testFolder, 0700)

	ts = testenv.StartTestServer(true)
	if err != nil {
		panic(err)
	}

	res := m.Run()
	ts.Stop()
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	t.Log("--- TestStartStop ---")

	svc := service.NewHiveovService(9999, true, nil, "",
		testCerts.ServerCert, testCerts.CaCert)
	hc1, _ := ts.AddConnectService(serviceID)
	err := svc.Start(hc1)
	require.NoError(t, err)
	time.Sleep(time.Second * 3)
	svc.Stop()
}
