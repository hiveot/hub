package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
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
	tdDoc1.AddAction(actionName, "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateTD(agentID, string(tddjson))
	require.NoError(t, err)

	// update the action
	err = dtwStore.UpdateActionStart(
		dThingID, actionName, actionValue, msgID, consumerID)
	require.NoError(t, err)

	// check progress
	v, err := svc.ValuesSvc.QueryAction(consumerID, digitwin.ValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    actionName})
	require.NoError(t, err)
	require.Equal(t, actionValue, v.Input)
	require.Equal(t, msgID, v.RequestID)

	// complete the action
	av, err := dtwStore.UpdateActionStatus(agentID, thingID, actionName,
		vocab.RequestCompleted, actionValue)
	require.NoError(t, err)
	require.Equal(t, msgID, av.RequestID)

	// check status
	v, err = svc.ValuesSvc.QueryAction(consumerID, digitwin.ValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    actionName})

	require.NoError(t, err)
	require.Equal(t, actionValue, v.Output)
	require.Equal(t, vocab.RequestCompleted, v.Progress)

	// read all actions
	actList, err := svc.ValuesSvc.QueryAllActions(consumerID, dThingID)
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
	err := svc.DirSvc.UpdateTD(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ValuesSvc.QueryAction("itsme", digitwin.ValuesQueryActionArgs{
		ThingID: "badthingid",
		Name:    "someevent"})
	assert.Error(t, err)
	// query non-existing action is allowed if strict is set to false
	_, err = svc.ValuesSvc.QueryAction("itsme", digitwin.ValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    "badeventname"})
	assert.NoError(t, err)
	_, err = svc.ValuesSvc.QueryAllActions("itsme", "badthingid")
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
	tdDoc1.AddAction(actionName, "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateTD(agentID, string(tddjson))
	require.NoError(t, err)

	// invoke the action with the wrong thing
	err = dtwStore.UpdateActionStart(
		"badThingID", actionName, actionValue, msgID, consumerID)
	assert.Error(t, err)

	// invoke the action with the wrong name
	err = dtwStore.UpdateActionStart(
		dThingID, "badName", actionValue, msgID, consumerID)
	assert.Error(t, err)

	// complete the action on wrong thing
	_, err = dtwStore.UpdateActionStatus(agentID, "badThingID", actionName,
		vocab.RequestPending, actionValue)
	assert.Error(t, err)

	// complete the action on wrong action name
	_, err = dtwStore.UpdateActionStatus(agentID, thingID, "badName",
		vocab.RequestCompleted, actionValue)
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
	tdDoc1.AddAction(actionName, "action 1", "", actionSchema)
	tddJSON1, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateTD(agentID, string(tddJSON1))
	require.NoError(t, err)
	tddJson2, err := svc.DirSvc.ReadTD(consumerID, dThingID)
	require.NoError(t, err)
	require.NotEmpty(t, tddJson2)

	// next, invoke the action to read the thing from the directory.
	ag := service.NewDigitwinAgent(svc)
	msg := hubclient.NewThingMessage(
		vocab.OpInvokeAction, digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, dThingID, consumerID)
	msg.CorrelationID = msgID
	stat := ag.HandleAction(msg)
	require.Empty(t, stat.Error)
	require.NotEmpty(t, stat.Output)
	require.Equal(t, vocab.RequestCompleted, stat.Status)

	// a non-existing DTD should fail
	msg = hubclient.NewThingMessage(
		vocab.OpInvokeAction, digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, "badid", consumerID)
	msg.CorrelationID = msgID
	stat = ag.HandleAction(msg)
	require.NotEmpty(t, stat.Error)

	// a non-existing method name should fail
	msg = hubclient.NewThingMessage(
		vocab.OpInvokeAction, digitwin.DirectoryDThingID, "badMethod", dThingID, consumerID)
	msg.CorrelationID = msgID
	stat = ag.HandleAction(msg)
	require.NotEmpty(t, stat.Error)

	// a non-existing serviceID should fail
	msg = hubclient.NewThingMessage(
		vocab.OpInvokeAction, "badservicename", digitwin.DirectoryReadTDMethod, dThingID, consumerID)
	msg.CorrelationID = msgID
	stat = ag.HandleAction(msg)
	require.NotEmpty(t, stat.Error)

}
