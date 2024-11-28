package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/hiveot/hub/wot/transports/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"testing"
)

var testDirFolder = path.Join(os.TempDir(), "test-directory")
var dirStorePath = path.Join(testDirFolder, "directory.data")
var cm *connections.ConnectionManager

// startService initializes a service and a client
// This doesn't use any transport.
func startService(clean bool) (
	svc *service.DigitwinService,
	store *store.DigitwinStore,
	stopFn func()) {

	if cm == nil {
		cm = connections.NewConnectionManager()
	}
	if clean {
		_ = os.RemoveAll(testDirFolder)
		cm.CloseAll()
	}
	svc, store, err := service.StartDigitwinService(dirStorePath, cm)
	if err != nil {
		panic("unable to start the digitwin service")
	}

	// use direct transport to pass messages to the service
	//msgHandler := digitwin.NewDirectoryHandler(svc)
	//cl = embedded.NewEmbeddedClient(digitwin.DirectoryAgentID, msgHandler)

	return svc, store, func() {
		svc.Stop()
	}
}

// generate a TD document with properties, events and actions
func createTDDoc(thingID string, nrProps, nrEvents, nrActions int) *tdd.TD {
	title := utils.CreateRandomName("title-", 0)
	td := tdd.NewTD(thingID, title, vocab.ThingDevice)
	for range nrProps {
		name := utils.CreateRandomName("prop-", 0)
		td.AddProperty(name, "", name, vocab.WoTDataTypeInteger)
	}
	for range nrEvents {
		name := utils.CreateRandomName("ev-", 0)
		td.AddEvent(name, name, "",
			&tdd.DataSchema{
				Type: vocab.WoTDataTypeInteger,
			})
	}
	for range nrActions {
		name := utils.CreateRandomName("act-", 0)
		td.AddAction(name, name, "",
			&tdd.DataSchema{
				Type: vocab.WoTDataTypeBool,
			})
	}
	return td
}

func TestStartStopService(t *testing.T) {
	var thingIDs = []string{"thing1", "thing2", "thing3", "thing4"}
	svc, hc, stopFunc := startService(true)
	_ = hc
	// add TDs
	for _, thingID := range thingIDs {
		td := createTDDoc(thingID, 1, 1, 1)
		tddjson, _ := json.Marshal(td)
		err := svc.DirSvc.UpdateTD("test", string(tddjson))
		require.NoError(t, err)
	}
	tds1, err := svc.ReadAllTDs("", 0, 10)
	require.NoError(t, err)
	require.Greater(t, len(tds1), 1)

	// stop and start again, the update should be reloaded
	stopFunc()

	svc, hc, stopFunc = startService(false)
	defer stopFunc()
	tds2, err := svc.ReadAllTDs("", 0, 10)
	assert.Equal(t, len(tds1), len(tds2))
}
