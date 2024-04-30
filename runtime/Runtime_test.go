package runtime_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/runtime"
	"github.com/hiveot/hub/runtime/api"
	"os"
	"path"
	"testing"
	"time"
)

const TestPort = 9444

var testDir = path.Join(os.TempDir(), "test-runtime")
var rtEnv = plugin.GetAppEnvironment(testDir, false)
var rtConfig = runtime.NewRuntimeConfig()
var certsBundle = certs.CreateTestCertBundle()

// start the runtime
func startRuntime() *runtime.Runtime {
	r := runtime.NewRuntime(rtConfig)
	err := r.Start(&rtEnv)
	if err != nil {
		panic("Failed to start runtime:" + err.Error())
	}
	return r
}

// Add a client and connect with its password
func addConnectClient(r *runtime.Runtime, clientType api.ClientType, clientID string) (cl *tlsclient.TLSClient, token string) {
	password := "pass1"
	err := r.AuthnSvc.AdminSvc.AddClient(clientType, clientID, clientID, "", password)
	if err != nil {
		panic("Failed adding client:" + err.Error())
	}

	addr := fmt.Sprintf("localhost:%d", TestPort)
	cl = tlsclient.NewTLSClient(addr, certsBundle.CaCert, time.Second*120)
	token, err = cl.ConnectWithPassword(clientID, password)
	if err != nil {
		panic("Failed connect with password:" + err.Error())
	}
	return cl, token
}

// create a TD with the given thingID
func createTD(thingID string) *things.TD {
	td := things.NewTD(thingID, "title-"+thingID, vocab.ThingSensor)
	return td
}

// TestMain for all authn tests, setup of default folders and filenames
func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	rtConfig.Protocols.HttpsBinding.Port = TestPort
	rtConfig.CaCert = certsBundle.CaCert
	rtConfig.CaKey = certsBundle.CaKey
	rtConfig.ServerKey = certsBundle.ServerKey
	rtConfig.ServerCert = certsBundle.ServerCert
	err := rtConfig.Setup(&rtEnv)
	if err != nil {
		panic("setup runtime config failed: " + err.Error())
	}

	_ = os.RemoveAll(testDir)

	res := m.Run()
	if res == 0 {
		_ = os.RemoveAll(testDir)
	}
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	r := startRuntime()
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}

func TestLogin(t *testing.T) {
	const senderID = "sender1"
	const password = "pass1"

	r := startRuntime()
	cl, _ := addConnectClient(r, api.ClientTypeUser, senderID)
	cl.Close()
	r.Stop()
	time.Sleep(time.Millisecond * 100)
}
