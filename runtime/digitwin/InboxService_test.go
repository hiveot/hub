package digitwin

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func startInboxService(clean bool) (*service.DigiTwinInboxService, func()) {
	_ = clean
	svc := service.NewDigiTwinInbox(nil, nil)
	err := svc.Start()
	if err != nil {
		panic("Failed to start the inbox service: " + err.Error())
	}
	return svc, func() {
		svc.Stop()
	}
}

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")

	res := m.Run()
	os.Exit(res)
}

func TestAddActionBadMsg(t *testing.T) {
	const msgType = vocab.MessageTypeAction
	const senderID = "User1"
	const thingID = "Agent1:Thing1"
	const key = "Key1"

	svc, stopFunc := startInboxService(true)
	defer stopFunc()

	// good params
	msg := things.NewThingMessage(msgType, thingID, key, nil, senderID)
	msg.MessageID = "t1"
	rec, err := svc.AddAction(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, rec)

	// duplicate message
	_, err = svc.AddAction(msg)
	assert.Error(t, err)

	// missing thingID
	msg = things.NewThingMessage(msgType, "", key, nil, senderID)
	msg.MessageID = "t1"
	rec, err = svc.AddAction(msg)
	assert.Error(t, err)
	// missing key
	msg = things.NewThingMessage(msgType, thingID, "", nil, senderID)
	msg.MessageID = "t1"
	rec, err = svc.AddAction(msg)
	assert.Error(t, err)
	// missing sender
	msg = things.NewThingMessage(msgType, thingID, key, nil, "")
	msg.MessageID = "t1"
	rec, err = svc.AddAction(msg)
	assert.Error(t, err)
	// missing messageID
	msg = things.NewThingMessage(msgType, thingID, key, nil, senderID)
	msg.MessageID = ""
	rec, err = svc.AddAction(msg)
	assert.Error(t, err)
}

func TestReadLatest(t *testing.T) {
	const msgType = vocab.MessageTypeAction
	const senderID = "User1"
	const thingID = "Agent1:Thing1"
	const key = "Key1"

	svc, stopFunc := startInboxService(true)
	defer stopFunc()

	msg := things.NewThingMessage(msgType, thingID, key, "data", senderID)
	msg.MessageID = "t1"
	rec, err := svc.AddAction(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, rec)

	args := digitwin.InboxReadLatestArgs{Key: key, ThingID: thingID}
	latest, err := svc.ReadLatest("", args)
	assert.NoError(t, err)
	assert.Equal(t, rec, latest)

	stopFunc()
}

func TestStartReadLatestWhenEmpty(t *testing.T) {
	const thingID = "Agent1:Thing1"
	const key = "Key1"

	svc, stopFunc := startInboxService(true)
	defer stopFunc()

	// read an empty store
	args := digitwin.InboxReadLatestArgs{
		Key:     key,
		ThingID: thingID,
	}
	latest, err := svc.ReadLatest("", args)
	assert.Error(t, err)
	assert.Empty(t, latest)
}

func TestUpdateDeliveryStatus(t *testing.T) {
	const msgType = vocab.MessageTypeAction
	const senderID = "User1"
	const thingID = "Agent1:Thing1"
	const key = "Key1"

	svc, stopFunc := startInboxService(true)
	defer stopFunc()

	msg := things.NewThingMessage(msgType, thingID, key, "data", senderID)
	msg.MessageID = "t1"
	rec, err := svc.AddAction(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, rec)

	// update the status to applied
	stat := hubclient.DeliveryStatus{}
	stat.Applied(msg)
	err = svc.AddDeliveryStatus(stat)
	require.NoError(t, err)

	// expect the status record to exist
	rec2, err := svc.GetRecord(msg.MessageID)
	require.NoError(t, err)
	require.Equal(t, rec.MessageID, rec2.MessageID)
	require.Equal(t, hubclient.DeliveryApplied, rec2.Progress)

	// update the status to completed
	stat = hubclient.DeliveryStatus{}
	stat.Completed(msg, "", nil)
	err = svc.AddDeliveryStatus(stat)
	require.NoError(t, err)

	// expect the status record to be gone but the 'latest' to be set to completed
	rec3, err := svc.GetRecord(msg.MessageID)
	require.Error(t, err)
	require.Empty(t, rec3)

	args := digitwin.InboxReadLatestArgs{Key: key, ThingID: thingID}
	latest, err := svc.ReadLatest("", args)
	assert.NoError(t, err)
	assert.Equal(t, hubclient.DeliveryCompleted, latest.Progress)

	stopFunc()
}

func TestBadDeliveryStatus(t *testing.T) {
	const msgType = vocab.MessageTypeAction
	const senderID = "User1"
	const thingID = "Agent1:Thing1"
	const key = "Key1"

	svc, stopFunc := startInboxService(true)
	defer stopFunc()

	msg := things.NewThingMessage(msgType, thingID, key, "data", senderID)
	msg.MessageID = "t1"
	rec, err := svc.AddAction(msg)
	assert.NoError(t, err)
	assert.NotEmpty(t, rec)

	stat := hubclient.DeliveryStatus{}
	err = svc.AddDeliveryStatus(stat)
	require.Error(t, err)

	// old status should remain
	args := digitwin.InboxReadLatestArgs{Key: key, ThingID: thingID}
	latest, err := svc.ReadLatest("", args)
	assert.NoError(t, err)
	assert.Equal(t, hubclient.DeliveredToInbox, latest.Progress)

	stopFunc()
}
