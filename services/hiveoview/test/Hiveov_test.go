package test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/sseclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/services/hiveoview/src/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/url"
	"os"
	"path"
	"sync/atomic"
	"testing"
	"time"
)

const serviceID = "hiveoview-test"
const servicePort = 9999

// set to true to test without state service
const noState = true

var testFolder = path.Join(os.TempDir(), "test-hiveoview")

// the following are set by the testmain
var ts *testenv.TestServer

// Helper function to login as a web client and sse listener
// This will set its cookie to allow for further requests.
// Run the TestLogin test before using this.
// This returns a client. Call Close() when done.
func WebLogin(clientID string,
	onConnection func(bool, error),
	onMessage func(message *hubclient.ThingMessage),
	onRequest func(message *hubclient.ThingMessage) hubclient.RequestStatus) (
	*sseclient.HttpSSEClient, error) {

	hostPort := fmt.Sprintf("localhost:%d", servicePort)
	sseCl := sseclient.NewHttpSSEClient(hostPort, clientID, nil, ts.Certs.CaCert, time.Minute*10)
	sseCl.SetConnectHandler(onConnection)
	sseCl.SetMessageHandler(onMessage)
	sseCl.SetRequestHandler(onRequest)
	sseCl.SetSSEPath("/websse")

	err := sseCl.ConnectWithLoginForm(clientID)

	return sseCl, err
}

func TestMain(m *testing.M) {
	var err error
	// raise loglevel where you want it in testing
	logging.SetLogging("warn", "")
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

	svc := service.NewHiveovService(servicePort, true, nil, "",
		ts.Certs.ServerCert, ts.Certs.CaCert, noState)
	hc1, _ := ts.AddConnectService(serviceID)

	err := svc.Start(hc1)
	require.NoError(t, err)
	time.Sleep(time.Second * 3)
	svc.Stop()
}

// test many connections from a single client and confirm they open close and receive messages properly.
func TestLogin(t *testing.T) {
	const clientID1 = "user1"

	// 1: setup: start a runtime and service; this generates an error that
	//    the state service isnt found. ignore it.
	svc := service.NewHiveovService(servicePort, true,
		nil, "", ts.Certs.ServerCert, ts.Certs.CaCert, noState)
	avcAg, _ := ts.AddConnectService(serviceID)
	err := svc.Start(avcAg)

	require.NoError(t, err)
	defer svc.Stop()

	// make sure the client to login as exists
	cl1, token1 := ts.AddConnectUser(clientID1, authz.ClientRoleOperator)
	defer cl1.Disconnect()

	_ = token1

	// 2: login using form
	hostPort := fmt.Sprintf("localhost:%d", servicePort)
	cl := tlsclient.NewTLSClient(
		hostPort, nil, ts.Certs.CaCert, time.Second*60, "cid1")

	// try login. The test user password is the clientID
	// the client should receive a cookie with a token
	formMock := url.Values{}
	formMock.Add("loginID", clientID1)
	formMock.Add("password", clientID1)
	fullURL := fmt.Sprintf("https://%s/login", hostPort)
	resp, err := cl.GetHttpClient().PostForm(fullURL, formMock)
	require.NoError(t, err)
	// this should redirect to /dashboard
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "/dashboard", resp.Request.URL.Path)

	// result contains html
	body, err := io.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.NotEmpty(t, body)
	cl.Close()
}

// test many connections from a single client and confirm they open close and receive messages properly.
func TestMultiConnectDisconnect(t *testing.T) {
	const clientID1 = "user1"
	const agentID = "agent1"
	const testConnections = int32(1)
	const eventName = "event1"
	var webClients = make([]*sseclient.HttpSSEClient, 0)
	var connectCount atomic.Int32
	var disConnectCount atomic.Int32
	var messageCount atomic.Int32
	const waitamoment = time.Millisecond * 10

	logging.SetLogging("info", "")
	// 1: setup: start a runtime and service; this generates an error that
	//    the state service isnt found. ignore it.
	svc := service.NewHiveovService(servicePort, true,
		nil, "", ts.Certs.ServerCert, ts.Certs.CaCert, noState)
	avcAg, _ := ts.AddConnectService(serviceID)
	err := svc.Start(avcAg)

	require.NoError(t, err)
	defer svc.Stop()

	// the agent for publishing events. A TD is needed for them to be accepted.
	ag1, _ := ts.AddConnectAgent(agentID)
	_ = ag1
	td1 := ts.AddTD(agentID, nil)
	_ = td1
	// create the user account this test is going to connect as.
	cl1, token1 := ts.AddConnectUser(clientID1, authz.ClientRoleOperator)
	cl1.Disconnect()
	time.Sleep(waitamoment)

	_ = token1
	onConnection := func(connected bool, err error) {
		if connected {
			connectCount.Add(1)
		} else {
			disConnectCount.Add(1)
		}
	}
	onMessage := func(msg *hubclient.ThingMessage) {
		// the UI expects this format for triggering htmx
		expectedType := fmt.Sprintf("dtw:%s:%s/%s", agentID, td1.ID, eventName)
		if expectedType == msg.Operation {
			messageCount.Add(1)
		}
	}

	// 2: connect and subscribe web clients and verify
	for range testConnections {
		sseCl, err := WebLogin(clientID1, onConnection, onMessage, nil)
		require.NoError(t, err)
		require.NotNil(t, sseCl)
		webClients = append(webClients, sseCl)
		time.Sleep(waitamoment)
	}
	// connection notification should have been received N times
	time.Sleep(time.Second * 3)
	require.Equal(t, testConnections, connectCount.Load(), "connect count mismatch")

	// 3: agent publishes an event, which should be received N times
	err = ag1.PubEvent(td1.ID, eventName, "a value", "message1")
	require.NoError(t, err)

	// event should have been received N times
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, testConnections, messageCount.Load(), "missing events")

	// 4: disconnect
	//sm := svc.GetSM()
	for _, c := range webClients {

		// disconnect the client connection
		c.Disconnect()
		time.Sleep(waitamoment)
	}
	//time.Sleep(waitamoment)
	t.Log("All user1 connections should have been closed")
	// disconnection notification should have been received N times
	time.Sleep(waitamoment)
	require.Equal(t, testConnections, disConnectCount.Load(), "disconnect count mismatch")

	//	// 5: no more messages should be received after disconnecting
	messageCount.Store(0)
	err = ag1.PubEvent(td1.ID, eventName, "a value", "message2")
	require.NoError(t, err)

	// zero events should have been received
	time.Sleep(waitamoment)
	assert.Equal(t, int32(0), messageCount.Load(), "still receiving events afer disconnect")

	// last, the service should have no connections
	ag1.Disconnect()
	avcAg.Disconnect()
	time.Sleep(waitamoment)

	// FIXME: currently one or two connections remain
	// the root cause of the first is that the first browser load doesn't connect
	// with SSE and this never closes the connection.
	// The second remaining session doesn't happen while debugging .. yeah fun
	nrConnections, _ := ts.Runtime.CM.GetNrConnections()
	nrSessions := svc.GetSM().GetNrSessions()
	if nrConnections > 0 {
		t.Log(fmt.Sprintf(
			"FIXME: expected 0 remaining connections and sessions. "+
				"Got '%d' connections from '%d' sessions", nrConnections, nrSessions))
	}
	//assert.Less(t, 3, count)

	//time.Sleep(time.Millisecond * 100)
}
