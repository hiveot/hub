package service_test

import (
	"github.com/hiveot/hub/api/go/thing"
	"github.com/hiveot/hub/core/service"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

var testCerts = testenv.CreateCertBundle()

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	res := m.Run()
	os.Exit(res)
}

func TestHubServer_Start(t *testing.T) {
	hs := service.NewHubServer(testCerts.ServerCert, testCerts.CaCert)
	clientURL, err := hs.Start()
	require.NoError(t, err)
	require.NotEmpty(t, clientURL)
	time.Sleep(time.Second * 2)

	hc := hubclient.NewHubClient("test1")
	err = hc.ConnectWithCert(clientURL, testCerts.DeviceID, testCerts.DeviceCert, testCerts.CaCert)
	assert.NoError(t, err)
	hc.DisConnect()

	hs.Stop()
}

func TestPubSub_AuthCert(t *testing.T) {
	rxchan := make(chan int)
	hs := service.NewHubServer(testCerts.ServerCert, testCerts.CaCert)
	clientURL, err := hs.Start()
	require.NoError(t, err)

	hc := hubclient.NewHubClient("test1")
	err = hc.ConnectWithCert(clientURL, testCerts.DeviceID, testCerts.DeviceCert, testCerts.CaCert)
	assert.NoError(t, err)

	err = hc.SubEvent("", "", "", func(tv *thing.ThingValue) {
		rxchan <- 1
	})
	assert.NoError(t, err)

	err = hc.PubEvent("thing1", "event1", []byte("hello"))
	assert.NoError(t, err)

	rxdata := <-rxchan
	assert.Equal(t, 1, rxdata)
	hc.DisConnect()
	hs.Stop()
}

func TestPubSub_AuthLogin(t *testing.T) {
	rxchan := make(chan int)
	hs := service.NewHubServer(testCerts.ServerCert, testCerts.CaCert)
	//hs := testenv.NewTestServer(testCerts.ServerCert, testCerts.CaCert)
	clientURL, err := hs.Start()
	defer hs.Stop()
	require.NoError(t, err)

	hc := hubclient.NewHubClient("test1")
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
