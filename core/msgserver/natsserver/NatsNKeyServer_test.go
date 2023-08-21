package natsserver_test

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
	"testing"
	"time"
)

var device1ID = "device1"
var device1Key, _ = nkeys.CreateUser()
var device1Pub, _ = device1Key.PublicKey()
var thing1ID = "thing1"

var user1ID = "user1"
var user1Pass = "pass1"
var user1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(user1Pass), 0)

var user2ID = "user2"
var user2Key, _ = nkeys.CreateUser()
var user2Pub, _ = user2Key.PublicKey()

var service1ID = "service1"
var service1Key, _ = nkeys.CreateUser()
var service1Pub, _ = service1Key.PublicKey()

var group1ID = "group1"
var group2ID = "group2"

// test users
var TestClients = []authn.AuthnEntry{
	{ClientProfile: authn.ClientProfile{
		ClientID:    device1ID,
		ClientType:  authn.ClientTypeDevice,
		DisplayName: "device1 1",
		PubKey:      device1Pub,
	}},
	{ClientProfile: authn.ClientProfile{
		ClientID:    user1ID,
		ClientType:  authn.ClientTypeUser,
		DisplayName: "user 1",
	}, PasswordHash: string(user1bcrypt),
	},
	{ClientProfile: authn.ClientProfile{
		ClientID:    user2ID,
		ClientType:  authn.ClientTypeUser,
		DisplayName: "user 2",
		PubKey:      user2Pub,
	}},
	{ClientProfile: authn.ClientProfile{
		ClientID:    service1ID,
		ClientType:  authn.ClientTypeService,
		DisplayName: "service 1",
		PubKey:      service1Pub,
	}},
}
var TestRoles = map[string]authz.RoleMap{
	user1ID:    {group1ID: authz.GroupRoleViewer},
	user2ID:    {group1ID: authz.GroupRoleOperator},
	service1ID: {group1ID: authz.GroupRoleManager, group2ID: authz.GroupRoleViewer},
	device1ID:  {group1ID: authz.GroupRoleIotDevice, group2ID: authz.GroupRoleIotDevice},
	thing1ID:   {group1ID: authz.GroupRoleThing, group2ID: authz.GroupRoleThing},
}

func TestStartStopNKeys(t *testing.T) {
	var rxMsg string

	clientURL, s, _, _, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)
	//err = s.ReloadClients(TestClients, TestRoles)
	//require.NoError(t, err)

	// connect using the built-in service key
	c, err := s.ConnectInProc("testnkeysservice", nil)
	require.NoError(t, err)
	require.NotEmpty(t, c)
	_, err = c.Subscribe("things.>", func(m *nats.Msg) {
		rxMsg = string(m.Data)
		slog.Info("received message", "msg", rxMsg)
	})
	require.NoError(t, err)
	err = c.Publish("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

	c.Close()
}

func TestConnectWithNKey(t *testing.T) {

	slog.Info("--- TestConnectWithNKey start")
	defer slog.Info("--- TestConnectWithNKey end")
	var rxMsg string

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ReloadClients(TestClients, TestRoles)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithNKey(
		clientURL, user2ID, user2Key, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	_, err = hc1.Sub("things.>", func(topic string, data []byte) {
		rxMsg = string(data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err)
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)

}
func TestConnectWithPassword(t *testing.T) {
	var rxMsg string
	slog.Info("--- TestConnectWithPassword start")
	defer slog.Info("--- TestConnectWithPassword end")

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ReloadClients(TestClients, TestRoles)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithPassword(
		clientURL, user1ID, user1Pass, certBundle.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	_, err = hc1.Sub("things.>", func(topic string, data []byte) {
		rxMsg = string(data)
		slog.Info("received message", "msg", rxMsg)
	})
	assert.NoError(t, err)
	err = hc1.Pub("things.service1.event", []byte("hello world"))
	require.NoError(t, err)
	time.Sleep(time.Millisecond)
	assert.Equal(t, "hello world", rxMsg)
}

// this requires the JWT server. It cannot be used together with NKeys :/
//func TestLoginWithJWT(t *testing.T) {
//	slog.Info("--- TestLoginWithJWT start")
//	defer slog.Info("--- TestLoginWithJWT end")
//
//	rxMsg := ""
//	_, stopFn, err := startTestAuthnService()
//	require.NoError(t, err)
//	defer stopFn()
//
//	// raw generate a jwt token
//	//userKey, _ := nkeys.CreateUser()
//	userKey := serverCfg.CoreServiceKP
//	userJWT := serverCfg.CoreServiceJWT
//	hc1, err := natshubclient.ConnectWithJWT(clientURL, userKey, userJWT, certBundle.CaCert)
//	require.NoError(t, err)
//
//	_, err = hc1.Subscribe("things.>", func(msg *nats.Msg) {
//		rxMsg = string(msg.Data)
//		slog.Info("received message", "msg", rxMsg)
//	})
//	assert.NoError(t, err, "unable to subscribe")
//	err = hc1.Pub("things.service1.event", []byte("hello world"))
//	require.NoError(t, err)
//	time.Sleep(time.Millisecond)
//	assert.Equal(t, "hello world", rxMsg)
//
//	hc1.Disconnect()
//}

//func TestLoginWithInvalidJWT(t *testing.T) {
//	slog.Info("--- TestLoginWithInvalidJWT start")
//	defer slog.Info("--- TestLoginWithInvalidJWT end")
//	_, stopFn, err := startTestAuthnService()
//	require.NoError(t, err)
//	defer stopFn()
//
//	// token signed by fake account should fail
//	fakeAccountKey, _ := nkeys.CreateAccount()
//	userKey, _ := nkeys.CreateUser()
//	userPub, _ := userKey.PublicKey()
//	userClaims := jwt.NewUserClaims(userPub)
//	userClaims.IssuerAccount, _ = fakeAccountKey.PublicKey()
//	badToken, _ := userClaims.Encode(fakeAccountKey)
//	hc1, err := natshubclient.ConnectWithJWT(clientURL, userKey, badToken, certBundle.CaCert)
//	require.Error(t, err)
//	require.Empty(t, hc1)
//
//}

func TestLoginFail(t *testing.T) {
	slog.Info("--- TestLoginFail start")
	defer slog.Info("--- TestLoginFail end")

	clientURL, s, _, certBundle, err := testenv.StartNatsTestServer()
	require.NoError(t, err)
	defer s.Stop()
	assert.NotEmpty(t, clientURL)

	// add several users, service and devices
	err = s.ReloadClients(TestClients, TestRoles)
	require.NoError(t, err)

	hc1, err := natshubclient.ConnectWithPassword(
		clientURL, user1ID, "wrongpassword", certBundle.CaCert)
	require.Error(t, err)

	// key doesn't belong to user
	//hc1, err = natshubclient.ConnectWithNKey(
	//	clientURL, user1ID, service1Key, certBundle.CaCert)
	//require.Error(t, err)

	_ = hc1
}
