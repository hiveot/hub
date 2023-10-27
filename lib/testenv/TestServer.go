package testenv

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authservice"
	"github.com/hiveot/hub/core/auth/config"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/hubclient/transports/mqtttransport"
	"github.com/hiveot/hub/lib/hubclient/transports/natstransport"
	"log/slog"
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
	testClients []msgserver.ClientAuthInfo // when using AddConnectClient
}

// AddClients adds test clients to the server.
// This either adds them to the server directly or adds them using auth.
func (ts *TestServer) AddClients(newClients []msgserver.ClientAuthInfo) error {
	var err error
	ctx := hubclient.ServiceContext{SenderID: "testServer"}
	if ts.AuthService != nil {
		for _, authInfo := range newClients {
			args := authapi.AddUserArgs{
				UserID:      authInfo.ClientID,
				DisplayName: authInfo.ClientID,
				PubKey:      authInfo.PubKey,
				Role:        authInfo.Role,
			}
			_, err = ts.AuthService.MngClients.AddUser(ctx, args)
			if err != nil {
				slog.Error("AddClients error", "clientID", authInfo.ClientID, "err", err)
			}
		}
	} else {
		ts.testClients = append(ts.testClients, newClients...)
		err = ts.MsgServer.ApplyAuth(ts.testClients)
	}
	return err
}

// AddConnectClient to the hub as a client type using the given clientID.
// This generates a key pair and auth token used to connect.
// Intended for easily connecting during tests to avoid a lot of auth boilerplate.
//
// If auth has started using the TestServer then this adds the client to auth service
// as a service. Don't use this method if auth is started separately. Use auth directly instead.
//
// Without auth service this applies it to the messaging server directly adding to
// what was set using AddClients() (don't use ApplyAuth on the message server directly)
//
//	agentID the connecting agent is required
//	clientType is optional. This defaults to ClientTypeUser.
//	clientRole is optional. This defaults to viewer.
func (ts *TestServer) AddConnectClient(
	agentID string, clientType string, clientRole string) (*hubclient.HubClient, error) {
	var token string
	var err error

	var tp transports.IHubTransport
	serverURL, _, _ := ts.MsgServer.GetServerURLs()
	if ts.Core == "nats" {
		tp = natstransport.NewNatsTransport(serverURL, agentID, ts.CertBundle.CaCert)
	} else {
		tp = mqtttransport.NewMqttTransport(serverURL, agentID, ts.CertBundle.CaCert)
	}
	serKP, serPub := tp.CreateKeyPair()

	if clientType == "" {
		clientType = authapi.ClientTypeUser
	}
	if clientRole == "" {
		clientRole = authapi.ClientRoleViewer
	}
	ctx := hubclient.ServiceContext{SenderID: "testServer"}

	// if auth service is running then add the user if it doesn't exist
	if ts.AuthService != nil {
		if clientType == authapi.ClientTypeService {
			args := authapi.AddServiceArgs{
				ServiceID:   agentID,
				DisplayName: "user " + agentID,
				PubKey:      serPub,
			}
			resp, err2 := ts.AuthService.MngClients.AddService(ctx, args)
			err = err2
			token = resp.Token
		} else if clientType == authapi.ClientTypeDevice {
			args := authapi.AddDeviceArgs{
				DeviceID:    agentID,
				DisplayName: "user " + agentID,
				PubKey:      serPub,
			}
			resp, err2 := ts.AuthService.MngClients.AddDevice(ctx, args)
			err = err2
			token = resp.Token
		} else {
			args := authapi.AddUserArgs{
				UserID:      agentID,
				DisplayName: "user " + agentID,
				PubKey:      serPub,
				Role:        clientRole,
			}
			resp, err2 := ts.AuthService.MngClients.AddUser(ctx, args)
			err = err2
			token = resp.Token
		}
	} else {
		// use an on-the-fly created token for the connection
		authInfo := msgserver.ClientAuthInfo{
			ClientID:     agentID,
			ClientType:   clientType,
			PubKey:       serPub,
			PasswordHash: "",
			Role:         clientRole,
		}
		token, err = ts.MsgServer.CreateToken(authInfo)
		if err == nil {
			ts.testClients = append(ts.testClients, authInfo)
			err = ts.MsgServer.ApplyAuth(ts.testClients)
		}
	}
	if err != nil {
		return nil, err
	}
	//safeConn := packets.NewThreadSafeConn(conn)
	//serverURL, _, _ := ts.MsgServer.GetServerURLs()
	//if ts.Core == "nats" {
	//	tp = natstransport.NewNatsTransport(serverURL, agentID, ts.CertBundle.CaCert)
	//} else {
	//	tp = mqtttransport.NewMqttTransport(serverURL, agentID, ts.CertBundle.CaCert)
	//}
	hc := hubclient.NewHubClientFromTransport(tp, agentID)
	err = hc.ConnectWithToken(serKP, token)

	return hc, err
}

// StartAuth starts the auth service
func (ts *TestServer) StartAuth() (err error) {
	var testDir = path.Join(os.TempDir(), "test-home")
	// clean start
	_ = os.RemoveAll(testDir)
	authConfig := config.AuthConfig{}
	_ = authConfig.Setup(testDir, testDir)
	ts.AuthService, err = authservice.StartAuthService(
		authConfig, ts.MsgServer, ts.CertBundle.CaCert)
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
// Use withAuth or run StartAuth() to start the auth service
func StartTestServer(core string, withAuth bool) (*TestServer, error) {
	var err error
	ts := &TestServer{
		CertBundle:  certs.CreateTestCertBundle(),
		Core:        core,
		testClients: make([]msgserver.ClientAuthInfo, 0),
	}
	if core == "nats" {
		ts.MsgServer, ts.CertBundle, _, err = StartNatsTestServer(false)
	} else {
		ts.MsgServer, ts.CertBundle, err = StartMqttTestServer()
	}
	if err == nil && withAuth {
		err = ts.StartAuth()
	}
	return ts, err
}
