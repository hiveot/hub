package natstransport_test

import (
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/mqtttransport"
	"github.com/hiveot/hub/runtime/transports/natstransport/callouthook"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestStartStopCallout(t *testing.T) {
	// defined in NatsNKeyServer_test.go
	s, certBundle, _, err := testenv.StartNatsTestServer(true)
	require.NoError(t, err)
	defer s.Stop()
	err = s.ApplyAuth(NatsTestClients)

	// enabling callout should succeed
	_, err = callouthook.EnableNatsCalloutHook(s)
	assert.NoError(t, err)

	// core services do not use the callout handler
	//c, err := s.ConnectInProc("testcalloutservice")
	serverURL, _, _ := s.GetServerURLs()
	tp := natstransport.NewNatsTransport(
		serverURL, "testcalloutservice", certBundle.CaCert)
	require.NotEmpty(t, tp)
	err = tp.ConnectWithKey(TestService1NKey)
	require.NoError(t, err)
	tp.Disconnect()
}

func TestValidateToken(t *testing.T) {
	t.Log("---TestToken start---")
	defer t.Log("---TestToken end---")

	// setup
	s, _, _, err := testenv.StartNatsTestServer(true)
	require.NoError(t, err)
	defer s.Stop()

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuth(NatsTestClients)
	assert.NoError(t, err)

	// enable callout so a jwt token can be generated
	_, err = callouthook.EnableNatsCalloutHook(s)
	assert.NoError(t, err)
	token, err := s.CreateToken(mqtttransport.ClientAuthInfo{
		ClientID:     TestDevice1ID,
		ClientType:   api.ClientTypeUser,
		PubKey:       TestDevice1NPub,
		PasswordHash: "",
		Role:         api.ClientRoleManager,
	})
	require.NoError(t, err)

	err = s.ValidateToken(TestDevice1ID, token, "", "")
	assert.NoError(t, err)
}

func TestCalloutPassword(t *testing.T) {
	s, _, certBundle, err := testenv.StartNatsTestServer(false)
	require.NoError(t, err)
	defer s.Stop()

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuth(NatsTestClients)
	assert.NoError(t, err)

	// this callout handler only accepts 'user2' request
	chook, err := callouthook.EnableNatsCalloutHook(s)
	assert.NoError(t, err)

	user1Key, _ := nkeys.CreateUser()
	serverURL, _, _ := s.GetServerURLs()
	tp1 := natstransport.NewNatsTransport(
		serverURL, TestUser1ID, certBundle.CaCert)
	_ = user1Key
	err = tp1.ConnectWithPassword(TestUser1Pass)
	require.NoError(t, err)
	successCount, _ := chook.GetCounters()
	assert.Equal(t, 1, successCount)
	tp1.Disconnect()
}

func TestCalloutJWT(t *testing.T) {
	s, _, certBundle, err := testenv.StartNatsTestServer(true)
	require.NoError(t, err)
	defer s.Stop()

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuth(NatsTestClients)
	assert.NoError(t, err)

	// this callout handler only accepts 'user2' request
	chook, err := callouthook.EnableNatsCalloutHook(s)
	assert.NoError(t, err)

	jwtToken, err := s.CreateJWTToken(msgserver_old.ClientAuthInfo{
		ClientID:     TestAdminUserID,
		ClientType:   authapi.ClientTypeUser,
		PubKey:       TestAdminUserNPub,
		PasswordHash: "",
		Role:         authapi.ClientRoleManager,
	})
	require.NoError(t, err)
	serverURL, _, _ := s.GetServerURLs()
	hc2 := natstransport.NewNatsTransport(serverURL, TestAdminUserID, certBundle.CaCert)
	err = hc2.ConnectWithJWT(TestAdminUserNKey, jwtToken)
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 10)
	successCount, failCount := chook.GetCounters()
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 0, failCount)

	hc2.Disconnect()
}

func TestNoCalloutForExistingNKey(t *testing.T) {
	s, certBundle, _, err := testenv.StartNatsTestServer(true)
	require.NoError(t, err)
	defer s.Stop()

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuth(NatsTestClients)
	assert.NoError(t, err)

	// a directly added service should not invoke the callout handler
	chook, err := callouthook.EnableNatsCalloutHook(s)
	assert.NoError(t, err)

	// (added by nkey server test)
	serverURL, _, _ := s.GetServerURLs()
	tp1 := natstransport.NewNatsTransport(
		serverURL, TestService1ID, certBundle.CaCert)
	err = tp1.ConnectWithKey(TestService1NKey)
	//c, err := s.ConnectInProc(testenv.TestService1ID)
	require.NoError(t, err)
	tp1.Disconnect()

	successCount, failCount := chook.GetCounters()
	assert.Equal(t, 0, successCount)
	assert.Equal(t, 0, failCount)
}

func TestInValidCalloutAuthn(t *testing.T) {
	const knownUser = "knownuser"

	s, _, certBundle, err := testenv.StartNatsTestServer(true)
	require.NoError(t, err)
	defer s.Stop()

	// add several predefined users, service and devices that don't need callout
	err = s.ApplyAuth(NatsTestClients)

	// this callout handler only accepts 'knownUser' request
	chook, err := callouthook.EnableNatsCalloutHook(s)
	_ = chook
	assert.NoError(t, err)

	// invoke callout by connecting with an invalid user
	newkey2, _ := nkeys.CreateUser()
	serverURL, _, _ := s.GetServerURLs()
	tp2 := natstransport.NewNatsTransport(serverURL, "unknownuser", certBundle.CaCert)
	err = tp2.ConnectWithKey(newkey2)
	require.Error(t, err)

	_, failCount := chook.GetCounters()
	assert.Equal(t, 1, failCount)

}
