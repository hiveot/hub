package digitwin_test

import (
	"encoding/json"
	digitwin2 "github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/runtime/digitwin/service"
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
	const msgID = "msg1"
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := &tdd.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "", "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateDTD(agentID, string(tddjson))
	require.NoError(t, err)

	// update the action
	err = dtwStore.UpdateActionStart(
		consumerID, dThingID, actionName, actionValue, msgID)
	require.NoError(t, err)

	// check progress
	v, err := svc.ValuesSvc.ReadAction(consumerID, digitwin2.ValuesReadActionArgs{
		ThingID: dThingID,
		Name:    actionName})
	require.NoError(t, err)
	require.Equal(t, actionValue, v.Input)

	// complete the action
	av, err := dtwStore.UpdateActionProgress(agentID, thingID, actionName,
		digitwin.StatusCompleted, actionValue)
	require.NoError(t, err)
	require.Equal(t, msgID, av.MessageID)

	// check status
	v, err = svc.ValuesSvc.ReadAction(consumerID, digitwin2.ValuesReadActionArgs{
		ThingID: dThingID,
		Name:    actionName})

	require.NoError(t, err)
	require.Equal(t, actionValue, v.Output)
	require.Equal(t, digitwin.StatusCompleted, v.Status)

	// read all actions
	actList, err := svc.ValuesSvc.ReadAllActions(consumerID, dThingID)
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
	err := svc.DirSvc.UpdateDTD(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ValuesSvc.ReadAction("itsme", digitwin2.ValuesReadActionArgs{
		ThingID: "badthingid",
		Name:    "someevent"})
	assert.Error(t, err)
	_, err = svc.ValuesSvc.ReadAction("itsme", digitwin2.ValuesReadActionArgs{
		ThingID: dThingID,
		Name:    "badeventname"})
	assert.Error(t, err)
	_, err = svc.ValuesSvc.ReadAllActions("itsme", "badthingid")
	assert.Error(t, err)
}

func TestInvokeActionErrors(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	const actionName = "action1"
	const actionValue = 25
	const msgID = "mid1"
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := &tdd.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "", "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateDTD(agentID, string(tddjson))
	require.NoError(t, err)

	// invoke the action with the wrong thing
	err = dtwStore.UpdateActionStart(
		consumerID, "badThingID", actionName, actionValue, msgID)
	assert.Error(t, err)

	// invoke the action with the wrong name
	err = dtwStore.UpdateActionStart(
		consumerID, dThingID, "badName", actionValue, msgID)
	assert.Error(t, err)

	// complete the action on wrong thing
	_, err = dtwStore.UpdateActionProgress(agentID, "badThingID", actionName,
		digitwin.StatusCompleted, actionValue)
	assert.Error(t, err)

	// complete the action on wrong action name
	_, err = dtwStore.UpdateActionProgress(agentID, thingID, "badName",
		digitwin.StatusCompleted, actionValue)
	assert.Error(t, err)
}

func TestDigitwinAgentAction(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	const actionName = "action1"
	const actionValue = 25
	const msgID = "mid1"
	dThingID := tdd.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := &tdd.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "", "action 1", "", actionSchema)
	tddJSON1, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateDTD(agentID, string(tddJSON1))
	require.NoError(t, err)
	tddJson2, err := svc.DirSvc.ReadDTD(consumerID, dThingID)
	require.NoError(t, err)
	require.NotEmpty(t, tddJson2)

	// next, invoke the action to read the thing from the directory.
	ag := service.NewDigitwinAgent(svc)
	status, output, err := ag.HandleAction(consumerID,
		digitwin2.DirectoryDThingID, digitwin2.DirectoryReadDTDMethod, dThingID, msgID)
	require.NoError(t, err)
	require.NotEmpty(t, output)
	require.Equal(t, digitwin.StatusCompleted, status)

	// last, a non-existing DTD should fail
	status, output, err = ag.HandleAction(consumerID,
		digitwin2.DirectoryDThingID, digitwin2.DirectoryReadDTDMethod, "badid", msgID)
	require.Error(t, err)
	// a non-existing method name should fail
	status, output, err = ag.HandleAction(consumerID,
		digitwin2.DirectoryDThingID, "badmethod", dThingID, msgID)
	require.Error(t, err)
	// a non-existing serviceID should fail
	status, output, err = ag.HandleAction(consumerID,
		"badservicename", digitwin2.DirectoryReadDTDMethod, dThingID, msgID)
	require.Error(t, err)

}
