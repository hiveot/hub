package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestAddReadEvent(t *testing.T) {
	const agent1ID = "agent1"
	const thing1ID = "thing1"
	const eventName = "event1"
	const eventValue = 25

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event
	tdDoc1 := createTDDoc(thing1ID, 5, 3, 1)
	tdDoc1.AddEvent(eventName, "event1", "Descr 1",
		&td.DataSchema{
			Title: "type1",
			Type:  vocab.WoTDataTypeInteger,
		})
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateTD(agent1ID, string(tdDoc1Json))

	// provide an event value
	dThingID, err := dtwStore.UpdateEventValue(agent1ID, thing1ID, eventName, eventValue, "")
	assert.NoError(t, err)

	// Read the event value and all events
	ev2, err := svc.ValuesSvc.ReadEvent("user1", digitwin.ValuesReadEventArgs{
		ThingID: dThingID,
		Name:    eventName,
	})
	assert.NoError(t, err)
	assert.Equal(t, eventValue, ev2.Data)
	evList, err := svc.ValuesSvc.ReadAllEvents("user1", dThingID)
	assert.Equal(t, 1, len(evList))
	assert.NoError(t, err)
}

func TestEventReadFail(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event

	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateTD(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ValuesSvc.ReadEvent("itsme", digitwin.ValuesReadEventArgs{
		ThingID: "badthingid",
		Name:    "someevent",
	})
	assert.Error(t, err)
	// if strict is set to false this can succeed with nil output
	_, err = svc.ValuesSvc.ReadEvent("itsme", digitwin.ValuesReadEventArgs{
		ThingID: dThingID,
		Name:    "badeventname",
	})
	assert.NoError(t, err)
	_, err = svc.ValuesSvc.ReadAllEvents("itsme", "badthingid")
	assert.Error(t, err)
}

func TestEventUpdateFail(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	const EventName = "event1"

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event

	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1.AddEvent(EventName, "event1", "Descr 1",
		&td.DataSchema{
			Title: "type1",
			Type:  vocab.WoTDataTypeInteger,
		})
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateTD(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = dtwStore.UpdateEventValue(agentID, "notathing", EventName, 123, "")
	assert.Error(t, err)

	//event names not in the TD are accepted
	dThingID2, err := dtwStore.UpdateEventValue(agentID, thingID, "notanevent", 123, "")
	assert.NoError(t, err)
	assert.NotEmpty(t, dThingID2)
}
