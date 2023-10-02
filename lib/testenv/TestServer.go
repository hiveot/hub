package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	authcfg "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl"
	"os"
	"path"
)

// TestServer for testing application services
// This server provides an easy way to connect to the server by automatically
// creating a user and auth token when using ConnectInProc.
type TestServer struct {
	Core        string
	CertBundle  certs.TestCertBundle
	MsgServer   msgserver.IMsgServer
	AuthService *authservice.AuthService
	ServerURL   string
}

// AddConnectClient to the hub as a client type using the given clientID.
// This generates a key pair and auth token used to connect.
// Intended for easily connecting during tests to avoid a lot of auth boilerplate.
// If auth has started then this adds the client to auth service as a service.
//
//	clientID is required
//	clientType is optional. This defaults to ClientTypeUser.
//	clientRole is optional. This defaults to viewer.
func (ts *TestServer) AddConnectClient(clientID string, clientType string, clientRole string) (hubclient.IHubClient, error) {
	var token string
	var err error

	kp, kpPub := ts.MsgServer.CreateKP()

	if clientType == "" {
		clientType = auth.ClientTypeUser
	}
	if clientRole == "" {
		clientRole = auth.ClientRoleViewer
	}

	// if auth service is running then add the user if it doesn't exist
	if ts.AuthService != nil {
		token, err = ts.AuthService.MngClients.AddUser(clientID, clientID, "", kpPub, clientRole)
	} else {
		// use an on-the-fly created token for the connection
		token, err = ts.MsgServer.CreateToken(msgserver.ClientAuthInfo{
			ClientID:     clientID,
			ClientType:   clientType,
			PubKey:       kpPub,
			PasswordHash: "",
			Role:         clientRole,
		})
	}
	if err != nil {
		return nil, err
	}
	//safeConn := packets.NewThreadSafeConn(conn)
	hc := hubcl.NewHubClient(ts.ServerURL, clientID, kp, ts.CertBundle.CaCert, ts.Core)
	err = hc.ConnectWithToken(token)

	return hc, err
}

// StartAuth starts the auth service
func (ts *TestServer) StartAuth() (err error) {
	var testDir = path.Join(os.TempDir(), "test-home")
	authConfig := authcfg.AuthConfig{}
	_ = authConfig.Setup(testDir, testDir)
	ts.AuthService, err = authservice.StartAuthService(authConfig, ts.MsgServer)
	return err
}

// Stop the test server and optionally the auth service
func (ts *TestServer) Stop() {
	if ts.AuthService != nil {
		ts.AuthService.Stop()
	}
	if ts.MsgServer != nil {
		ts.MsgServer.Stop()
	}
}

// StartTestServer creates a NATS or MQTT test server depending on the requested type
// core is either "nats", or "mqtt" (default)
// This generates a certificate bundle for running the server, including a self signed CA.
//
// Use Stop() to clean up.
// Use StartAuth() to start the auth service
func StartTestServer(core string) (*TestServer, error) {
	var err error
	ts := &TestServer{
		CertBundle: certs.CreateTestCertBundle(),
		Core:       core,
	}
	if core == "nats" {
		ts.ServerURL, ts.MsgServer, ts.CertBundle, _, err = StartNatsTestServer(false)
	} else {
		ts.ServerURL, ts.MsgServer, ts.CertBundle, err = StartMqttTestServer()
	}

	return ts, err
}
