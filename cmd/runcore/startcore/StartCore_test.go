package startcore_test

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/cmd/runcore/startcore"
	"github.com/hiveot/hub/core/config"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/natsmsgserver"
	"github.com/hiveot/hub/lib/logging"
	"github.com/nats-io/nkeys"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"path"
	"testing"
	"time"
)

// var authBundle = testenv.CreateTestAuthBundle()
// var tempFolder = ""
var hubCfg *config.HubCoreConfig

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	homeDir := path.Join(os.TempDir(), "test-core")

	hubCfg = config.NewHubCoreConfig()
	hubCfg.NatsServer.Port = 9998
	// clear all existing data if any
	err := hubCfg.Setup(homeDir, "", true)
	if err != nil {
		slog.Error(err.Error())
	}

	res := m.Run()
	os.Exit(res)
}

func TestHubServer_StartStop(t *testing.T) {
	_ = os.Remove(hubCfg.Auth.PasswordFile)
	clientURL, err := startcore.Start(hubCfg)
	require.NoError(t, err)
	defer startcore.Stop()

	assert.NotEmpty(t, clientURL)
	time.Sleep(time.Second * 1)
}

func TestPubSub_ConnectAuthNKey(t *testing.T) {
	device1ID := "device1"
	device1KP, _ := nkeys.CreateUser()
	device1Pub, _ := device1KP.PublicKey()
	service1ID := "service1"
	service1KP, _ := nkeys.CreateUser()
	service1Pub, _ := service1KP.PublicKey()
	rxchan := make(chan int)
	clientURL := ""

	_ = os.Remove(hubCfg.Auth.PasswordFile)
	clientURL, err := startcore.Start(hubCfg)
	require.NoError(t, err)
	defer startcore.Stop()

	// setup: device and service
	_, err = startcore.AuthService.MngClients.AddDevice(device1ID, "device 1", device1Pub)
	require.NoError(t, err)
	_, err = startcore.AuthService.MngClients.AddService(service1ID, "service 1", service1Pub)
	require.NoError(t, err)

	// service1 subscribes to events
	hc1, err := natshubclient.ConnectWithNKey(clientURL, service1ID, service1KP, hubCfg.NatsServer.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	sub, err := hc1.SubStream(natsmsgserver.EventsIntakeStreamName, false, func(msg *hubclient.EventMessage) {
		slog.Info("received event", "id", msg.EventID)
		rxchan <- 1
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	hc2, err := natshubclient.ConnectWithNKey(clientURL, device1ID, device1KP, hubCfg.NatsServer.CaCert)
	err = hc2.PubEvent("thing1", "event1", []byte("hello"))
	assert.NoError(t, err)

	rxdata := <-rxchan
	assert.Equal(t, 1, rxdata)
	//time.Sleep(time.Second * 1)
}

func TestPubSub_AuthPassword(t *testing.T) {
	clientURL := ""
	rxchan := make(chan int)
	device1ID := "device1"
	device1KP, _ := nkeys.CreateUser()
	device1Pub, _ := device1KP.PublicKey()
	thing1ID := "thing1"
	user1ID := "user1"
	user1Pass := "user1pass"
	service1ID := "service1"
	service1KP, _ := nkeys.CreateUser()
	service1Pub, _ := service1KP.PublicKey()

	// launch the core services
	_ = os.Remove(hubCfg.Auth.PasswordFile)
	clientURL, err := startcore.Start(hubCfg)
	require.NoError(t, err)
	defer startcore.Stop()

	// add a device, service and user to test with
	_, err = startcore.AuthService.MngClients.AddUser(user1ID, "u 1", user1Pass, "", auth.ClientRoleViewer)
	require.NoError(t, err)
	_, err = startcore.AuthService.MngClients.AddService(service1ID, "s 1", service1Pub)
	require.NoError(t, err)
	_, err = startcore.AuthService.MngClients.AddDevice(device1ID, "d 1", device1Pub)
	require.NoError(t, err)

	// connect the user
	//hc1, err := natscoreclient.ConnectWithNKey(clientURL, service1ID, service1KP, hubCfg.NatsServer.CaCert)
	hc1, err := natshubclient.ConnectWithPassword(clientURL, user1ID, user1Pass, hubCfg.NatsServer.CaCert)
	require.NoError(t, err)
	defer hc1.Disconnect()

	si, err := hc1.JS().StreamInfo(natsmsgserver.EventsIntakeStreamName)
	assert.NoError(t, err)
	assert.NotEmpty(t, si)

	// listen for events
	sub, err := hc1.SubStream(natsmsgserver.EventsIntakeStreamName, false, func(msg *hubclient.EventMessage) {
		slog.Info("received event", "id", msg.EventID)
		rxchan <- 1
	})
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// connect the device
	hc2, err := natshubclient.ConnectWithNKey(clientURL, device1ID, device1KP, hubCfg.NatsServer.CaCert)
	require.NoError(t, err)
	defer hc2.Disconnect()
	err = hc2.PubEvent(thing1ID, "event1", []byte("hello"))
	assert.NoError(t, err)

	rxdata := <-rxchan
	assert.Equal(t, 1, rxdata)
}

//func TestPubSub_AuthJWT(t *testing.T) {
//	rxchan := make(chan int)
//	clientURL := ""
//
//	// launch the core services
//	core := natscorecore.NewHubCore()
//	require.NotPanics(t, func() { clientURL = core.Start(hubCfg) })
//	defer core.Stop()
//
//	// use the tokenizer to create a service token
//	serviceID := "service1"
//	serviceKey, _ := nkeys.CreateUser()
//	serviceKeyPub, _ := serviceKey.PublicKey()
//	tokenizer := nkeyserver.NewNatsAuthnTokenizer(hubCfg.NatsServer.AppAccountKP, true)
//	serviceJWT, err := tokenizer.CreateToken(serviceID, authn.ClientTypeService, serviceKeyPub, 0)
//	require.NoError(t, err)
//
//	// connect using the JWT token
//	hc, err := natscoreclient.ConnectWithJWT(clientURL, serviceKey, serviceJWT, hubCfg.NatsServer.CaCert)
//	require.NoError(t, err)
//	defer hc.Disconnect()
//
//	sub, err := hc.SubEvents("", func(msg *hubclient.EventMessage) {
//		slog.Info("received event", "eventID", msg.EventID)
//		rxchan <- 1
//	})
//	assert.NotEmpty(t, sub)
//	assert.NoError(t, err)
//
//	err = hc.PubEvent("thing1", "event1", []byte("hello"))
//	assert.NoError(t, err)
//
//	rxdata := <-rxchan
//	assert.Equal(t, 1, rxdata)
//	//time.Sleep(time.Second * 1)
//}

//func TestHubServer_Groups(t *testing.T) {
//	var rxcount1 atomic.Int32
//	var rxcount2 atomic.Int32
//	rxchan1 := make(chan int)
//	rxchan2 := make(chan int)
//
//	srv := hub.NewHubServer()
//	clientURL, err := srv.Start("", 0, testCerts.ServerCert, testCerts.CaCert)
//	defer srv.Stop()
//	require.NoError(t, err)
//	require.NotEmpty(t, clientURL)
//	time.Sleep(time.Millisecond * 100)
//
//	// add the device using its nkey public key
//	deviceUser, _ := testCerts.DeviceKey.PublicKey()
//	err = srv.AddUser(deviceUser, allPermissions)
//	assert.NoError(t, err)
//	defer srv.Stop()
//
//	hc := hubconn.NewHubClient("test1")
//	err = hc.ConnectWithNKey(clientURL, testCerts.DeviceKey, testCerts.CaCert)
//	require.NoError(t, err)
//	defer hc.DisConnect()
//
//	err = hc.DeleteGroup("events")
//	err = hc.DeleteGroup("group1")
//	err = hc.DeleteGroup("group2")
//	assert.NoError(t, err)
//
//	// add the ingress stream that receives all events
//	err = hc.AddStream("events", []string{"things.*.*.event.>"})
//	assert.NoError(t, err)
//
//	// add two group streams that receives events from from the ingress stream
//	// each group has a filter on the things that are a member of the group
//	err = hc.CreateGroup("group1", "events",
//		[]string{"things.*.thing1.event.>"})
//	assert.NoError(t, err)
//
//	err = hc.CreateGroup("group2", "events",
//		[]string{"things.*.thing2.event.>", "things.*.thing3.event.>"})
//	assert.NoError(t, err)
//
//	// group 1 should only receive events from thing1
//	err = hc.SubGroup("group1", func(tv *thing.ThingValue) {
//		slog.Info("received group 1 event", "thingID", tv.ThingID, "eventID", tv.ID)
//		rxcount1.Add(1)
//		rxchan1 <- 1
//	})
//	assert.NoError(t, err)
//	// group 2 should receive events from both thing2 and thing3
//	err = hc.SubGroup("group2", func(tv *thing.ThingValue) {
//		slog.Info("received group 2 event", "thingID", tv.ThingID, "eventID", tv.ID)
//		rxcount2.Add(1)
//		rxchan2 <- 2
//	})
//	assert.NoError(t, err)
//
//	err = hc.PubEvent("thing1", "event-A", []byte("hello"))
//	err = hc.PubEvent("thing2", "event-B", []byte("world"))
//	err = hc.PubEvent("thing3", "event-C", []byte("oh 3"))
//	assert.NoError(t, err)
//
//	<-rxchan1
//	<-rxchan2
//	time.Sleep(time.Millisecond * 100)
//	assert.Equal(t, int32(1), rxcount1.Load())
//	assert.Equal(t, int32(2), rxcount2.Load())
//}
