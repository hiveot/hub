package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/tdd"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestActionFlow(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	const actionName = "action1"
	const actionValue = 25
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := tdd.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "", "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.UpdateTD(agentID, thingID, string(tddjson))
	require.NoError(t, err)

	// invoke the action
	_, status, err = svc.InvokeAction(consumerID, dThingID, actionName, actionValue)
	require.NoError(t, err)

	// check progress
	act, err := svc.ReadAction(consumerID, dThingID, actionName)
	require.NoError(t, err)
	require.Equal(t, actionValue, act.Input)

	// complete the action
	err = svc.UpdateActionProgress(agentID, thingID, actionName,
		digitwin.StatusCompleted, actionValue)
	require.NoError(t, err)

	// check status
	act, err = svc.ReadAction(consumerID, dThingID, actionName)
	require.NoError(t, err)
	require.Equal(t, actionValue, act.Output)
	require.Equal(t, digitwin.StatusCompleted, act.Status)

	// read all actions
	actList, err := svc.ReadAllActions(consumerID, dThingID)
	require.NoError(t, err)
	require.NotZero(t, len(actList))
}

func TestActionReadFail(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	var dThingID = tdd.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an action
	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.UpdateTD(agentID, thingID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ReadAction("itsme", "badthingid", "someevent")
	assert.Error(t, err)
	_, err = svc.ReadAction("itsme", dThingID, "badeventname")
	assert.Error(t, err)
	_, err = svc.ReadAllActions("itsme", "badthingid")
	assert.Error(t, err)
}

func TestInvokeActionErrors(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	const actionName = "action1"
	const actionValue = 25
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := tdd.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "", "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.UpdateTD(agentID, thingID, string(tddjson))
	require.NoError(t, err)

	// invoke the action with the wrong thing
	err = svc.InvokeAction(consumerID, "badThingID", actionName, actionValue)
	assert.Error(t, err)

	// invoke the action with the wrong name
	err = svc.InvokeAction(consumerID, dThingID, "badName", actionValue)
	assert.Error(t, err)

	// complete the action on wrong thing
	err = svc.UpdateActionProgress(agentID, "badThingID", actionName,
		digitwin.ActionStatusCompleted, actionValue)
	assert.Error(t, err)

	// complete the action on wrong action name
	err = svc.UpdateActionProgress(agentID, thingID, "badName",
		digitwin.ActionStatusCompleted, actionValue)
	assert.Error(t, err)
}
