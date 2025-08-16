package digitwin_test

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"testing"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/connections"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testDirFolder = path.Join(os.TempDir(), "test-directory")
var dirStorePath = path.Join(testDirFolder, "directory.data")
var cm *connections.ConnectionManager

// startService initializes a service and a client
// This doesn't use any transport.
func startService(clean bool) (
	svc *service.DigitwinService,
	dirStore *store.DigitwinStore,
	stopFn func()) {

	if cm == nil {
		cm = connections.NewConnectionManager()
	}
	if clean {
		_ = os.RemoveAll(testDirFolder)
		cm.CloseAll()
	}
	notifHandler := func(notif *messaging.NotificationMessage) {
		slog.Info("Received notification", "op", notif.Operation)
	}
	svc, dirStore, err := service.StartDigitwinService(dirStorePath, notifHandler, true)
	if err != nil {
		panic("unable to start the digitwin service")
	}

	// use direct transport to pass messages to the service
	//msgHandler := digitwin.NewDirectoryHandler(svc)
	//cl = embedded.NewEmbeddedClient(digitwin.DirectoryAgentID, msgHandler)

	return svc, dirStore, func() {
		svc.Stop()
	}
}

// generate a TD document with properties, events and actions
func createTDDoc(thingID string, nrProps, nrEvents, nrActions int) *td.TD {
	title := utils.CreateRandomName("title-", 0)
	tdi := td.NewTD(thingID, title, vocab.ThingDevice)
	for range nrProps {
		name := utils.CreateRandomName("prop-", 0)
		tdi.AddProperty(name, "", name, vocab.WoTDataTypeInteger)
	}
	for range nrEvents {
		name := utils.CreateRandomName("ev-", 0)
		tdi.AddEvent(name, name, "",
			&td.DataSchema{
				Type: vocab.WoTDataTypeInteger,
			})
	}
	for range nrActions {
		name := utils.CreateRandomName("act-", 0)
		tdi.AddAction(name, name, "",
			&td.DataSchema{
				Type: vocab.WoTDataTypeBool,
			})
	}
	return tdi
}

func TestStartStopService(t *testing.T) {
	t.Log(fmt.Sprintf("---%s---\n", t.Name()))
	var thingIDs = []string{"thing1", "thing2", "thing3", "thing4"}
	svc, hc, stopFunc := startService(true)
	_ = hc
	// add TDs
	for _, thingID := range thingIDs {
		tdi := createTDDoc(thingID, 1, 1, 1)
		tddjson, _ := json.Marshal(tdi)
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
