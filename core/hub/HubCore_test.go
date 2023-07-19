package hub_test

import (
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

var testCerts = testenv.CreateTestAuthBundle()

var thingsPermissions = &server.Permissions{
	Publish:   &server.SubjectPermission{Allow: []string{"things.>"}, Deny: []string{"other.>"}},
	Subscribe: &server.SubjectPermission{Allow: []string{"things.>"}, Deny: []string{"other.>"}},
	//Publish:   &server.SubjectPermission{Allow: []string{">"}},
	//Subscribe: &server.SubjectPermission{Allow: []string{">"}},
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	res := m.Run()
	os.Exit(res)
}

func TestHubServer_Start(t *testing.T) {
	srv := hub.NewHubServer()
	clientURL, err := srv.Start("", 0, testCerts.ServerCert, testCerts.CaCert)
	require.NoError(t, err)
	require.NotEmpty(t, clientURL)
	time.Sleep(time.Second * 2)

	hc := hubconn.NewHubClient("test1")
	err = hc.ConnectWithCert(clientURL, testCerts.ServiceID, testCerts.ServerCert, testCerts.CaCert)
	assert.NoError(t, err)
	hc.DisConnect()

	srv.Stop()
}

func TestPubSub_AuthNKey(t *testing.T) {
	rxchan := make(chan int)
	srv := hub.NewHubServer()
	clientURL, err := srv.Start("", 0, testCerts.ServerCert, testCerts.CaCert)
	require.NoError(t, err)

	// add the device using its nkey public key
	deviceUser, _ := testCerts.DeviceNKey.PublicKey()
	err = srv.AddUser(deviceUser, thingsPermissions)
	assert.NoError(t, err)
	defer srv.Stop()

	hc := hubconn.NewHubClient("test1")
	err = hc.ConnectWithNKey(clientURL, testCerts.DeviceNKey, testCerts.CaCert)
	defer hc.DisConnect()

	assert.NoError(t, err)

	err = hc.SubEvent("", "", "", func(tv *thing.ThingValue) {
		slog.Info("received event", "id", tv.ID)
		rxchan <- 1
	})
	assert.NoError(t, err)

	err = hc.PubEvent("thing1", "event1", []byte("hello"))
	assert.NoError(t, err)

	rxdata := <-rxchan
	assert.Equal(t, 1, rxdata)
	//time.Sleep(time.Second * 1)
}

func TestPubSub_AuthPassword(t *testing.T) {
	rxchan := make(chan int)
	srv := hub.NewHubServer()
	clientURL, err := srv.Start("", 0, testCerts.ServerCert, testCerts.CaCert)
	defer srv.Stop()
	require.NoError(t, err)

	hc := hubconn.NewHubClient("test1")
	defer hc.DisConnect()
	err = hc.ConnectWithPassword(
		clientURL, "user1", "pass1", testCerts.CaCert)
	require.NoError(t, err)

	err = hc.SubEvent("", "", "", func(tv *thing.ThingValue) {
		rxchan <- 1
	})
	require.NoError(t, err)

	err = hc.PubEvent("thing1", "event1", []byte("hello"))
	assert.NoError(t, err)

	rxdata := <-rxchan
	assert.Equal(t, 1, rxdata)
}

func TestPubSub_AuthJWT(t *testing.T) {
	rxchan := make(chan int)
	srv := hub.NewHubServer()
	clientURL, err := srv.Start("", 0, testCerts.ServerCert, testCerts.CaCert)
	require.NoError(t, err)

	// add the device using its nkey public key
	//deviceUser, _ := testCerts.DeviceNKey.PublicKey()
	//err = srv.AddUser(deviceUser)
	assert.NoError(t, err)
	defer srv.Stop()

	hc := hubconn.NewHubClient("test1")
	err = hc.ConnectWithJWT(clientURL, testCerts.DeviceCreds, testCerts.CaCert)
	defer hc.DisConnect()

	assert.NoError(t, err)

	err = hc.SubEvent("", "", "", func(tv *thing.ThingValue) {
		slog.Info("received event", "id", tv.ID)
		rxchan <- 1
	})
	assert.NoError(t, err)

	err = hc.PubEvent("thing1", "event1", []byte("hello"))
	assert.NoError(t, err)

	rxdata := <-rxchan
	assert.Equal(t, 1, rxdata)
	//time.Sleep(time.Second * 1)
}

func TestHubServer_Groups(t *testing.T) {
	var rxcount1 atomic.Int32
	var rxcount2 atomic.Int32
	rxchan1 := make(chan int)
	rxchan2 := make(chan int)

	srv := hub.NewHubServer()
	clientURL, err := srv.Start("", 0, testCerts.ServerCert, testCerts.CaCert)
	defer srv.Stop()
	require.NoError(t, err)
	require.NotEmpty(t, clientURL)
	time.Sleep(time.Millisecond * 100)

	// add the device using its nkey public key
	deviceUser, _ := testCerts.DeviceNKey.PublicKey()
	err = srv.AddUser(deviceUser, allPermissions)
	assert.NoError(t, err)
	defer srv.Stop()

	hc := hubconn.NewHubClient("test1")
	err = hc.ConnectWithNKey(clientURL, testCerts.DeviceNKey, testCerts.CaCert)
	require.NoError(t, err)
	defer hc.DisConnect()

	err = hc.DeleteGroup("events")
	err = hc.DeleteGroup("group1")
	err = hc.DeleteGroup("group2")
	assert.NoError(t, err)

	// add the ingress stream that receives all events
	err = hc.AddStream("events", []string{"things.*.*.event.>"})
	assert.NoError(t, err)

	// add two group streams that receives events from from the ingress stream
	// each group has a filter on the things that are a member of the group
	err = hc.AddGroup("group1", "events",
		[]string{"things.*.thing1.event.>"})
	assert.NoError(t, err)

	err = hc.AddGroup("group2", "events",
		[]string{"things.*.thing2.event.>", "things.*.thing3.event.>"})
	assert.NoError(t, err)

	// group 1 should only receive events from thing1
	err = hc.SubGroup("group1", func(tv *thing.ThingValue) {
		slog.Info("received group 1 event", "thingID", tv.ThingID, "eventID", tv.ID)
		rxcount1.Add(1)
		rxchan1 <- 1
	})
	assert.NoError(t, err)
	// group 2 should receive events from both thing2 and thing3
	err = hc.SubGroup("group2", func(tv *thing.ThingValue) {
		slog.Info("received group 2 event", "thingID", tv.ThingID, "eventID", tv.ID)
		rxcount2.Add(1)
		rxchan2 <- 2
	})
	assert.NoError(t, err)

	err = hc.PubEvent("thing1", "event-A", []byte("hello"))
	err = hc.PubEvent("thing2", "event-B", []byte("world"))
	err = hc.PubEvent("thing3", "event-C", []byte("oh 3"))
	assert.NoError(t, err)

	<-rxchan1
	<-rxchan2
	time.Sleep(time.Millisecond * 100)
	assert.Equal(t, int32(1), rxcount1.Load())
	assert.Equal(t, int32(2), rxcount2.Load())
}
