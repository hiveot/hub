package digitwin_test

import (
	"encoding/json"
	"testing"

	"github.com/hiveot/gocore/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddReadEvent(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agent1ID = "agent1"
	const thing1ID = "thing1"
	const eventName = "event1"
	const eventValue = 25
	dThing1ID := td.MakeDigiTwinThingID(agent1ID, thing1ID)

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
	err := svc.DirSvc.UpdateThing(agent1ID, string(tdDoc1Json))

	// provide an event value
	evVal := digitwin.ThingValue{Data: eventValue, Name: eventName, ThingID: dThing1ID}
	err = dtwStore.UpdateEventValue(evVal)
	assert.NoError(t, err)

	// Read the event value and all events
	ev2, err := svc.ValuesSvc.ReadEvent("user1", digitwin.ThingValuesReadEventArgs{
		ThingID: dThing1ID,
		Name:    eventName,
	})
	assert.NoError(t, err)
	assert.Equal(t, eventValue, ev2.Data)
	evList, err := svc.ValuesSvc.ReadAllEvents("user1", dThing1ID)
	assert.Equal(t, 1, len(evList))
	assert.NoError(t, err)
}

func TestEventReadFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const thingID = "thing1"
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an event

	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ValuesSvc.ReadEvent("itsme", digitwin.ThingValuesReadEventArgs{
		ThingID: "badthingid",
		Name:    "someevent",
	})
	assert.Error(t, err)
	// if strict is set to false this can succeed with nil output
	_, err = svc.ValuesSvc.ReadEvent("itsme", digitwin.ThingValuesReadEventArgs{
		ThingID: dThingID,
		Name:    "badeventname",
	})
	assert.NoError(t, err)
	_, err = svc.ValuesSvc.ReadAllEvents("itsme", "badthingid")
	assert.Error(t, err)
}

func TestEventUpdateFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
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
	err := svc.DirSvc.UpdateThing(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	evVal := digitwin.ThingValue{Data: 123, Name: EventName, ThingID: "notathing"}
	err = dtwStore.UpdateEventValue(evVal)
	assert.Error(t, err)

	//event names not in the TD are accepted
	dThingID2 := td.MakeDigiTwinThingID(agentID, thingID)
	evVal = digitwin.ThingValue{Data: 123, Name: "notanevent", ThingID: dThingID2}
	err = dtwStore.UpdateEventValue(evVal)
	assert.NoError(t, err)
	assert.NotEmpty(t, dThingID2)
}
