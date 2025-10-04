package digitwin_test

import (
	"encoding/json"
	"testing"

	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionFlow(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const thingID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	const actionName = "action1"
	const actionValue = 25
	const correlationID = "req-1"
	dThingID := td.MakeDigiTwinThingID(agentID, thingID)

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := &td.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agentID, string(tddjson))
	require.NoError(t, err)

	// create the action
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, dThingID, actionName, actionValue, correlationID)
	as, stored, err := dtwStore.NewActionStart(req)
	require.NoError(t, err)
	require.True(t, stored)
	require.Equal(t, messaging.StatusPending, as.State)

	// check progress
	as, err = svc.ValuesSvc.QueryAction(consumerID, digitwin.ThingValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    actionName})
	require.NoError(t, err)
	inputVal := tputils.DecodeAsInt(as.Input)
	require.Equal(t, actionValue, inputVal)
	require.Equal(t, correlationID, as.ActionID)
	require.Equal(t, messaging.StatusPending, as.State)

	// complete the action
	resp := messaging.NewResponseMessage(
		wot.OpInvokeAction, dThingID, actionName, actionValue, nil, correlationID)
	as, err = dtwStore.UpdateActionWithResponse(resp)
	require.NoError(t, err)
	require.Equal(t, correlationID, as.ActionID)
	assert.Equal(t, "completed", as.State)

	// read action status
	as, err = svc.ValuesSvc.QueryAction(consumerID, digitwin.ThingValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    actionName})

	require.NoError(t, err)
	outputInt := tputils.DecodeAsInt(as.Output)
	require.Equal(t, actionValue, outputInt)
	require.Equal(t, messaging.StatusCompleted, as.State)

	// read all actions
	//actList, err := svc.ValuesSvc.QueryAllActions(consumerID, dThingID)
	//require.NoError(t, err)
	//require.NotZero(t, len(actList))
}

func TestActionReadFail(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const thingID = "thing1"
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// add a TD with an action
	tdDoc1 := createTDDoc(thingID, 4, 2, 1)
	tdDoc1Json, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agentID, string(tdDoc1Json))
	require.NoError(t, err)

	_, err = svc.ValuesSvc.QueryAction("itsme", digitwin.ThingValuesQueryActionArgs{
		ThingID: "badthingid",
		Name:    "someevent"})
	assert.Error(t, err)

	// query non-existing action is allowed if strict is set to false
	_, err = svc.ValuesSvc.QueryAction("itsme", digitwin.ThingValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    "badeventname"})
	require.NoError(t, err)
}

func TestInvokeActionErrors(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const thingID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	const actionName = "action1"
	const actionValue = 25
	const correlationID = "request-1"
	dThingID := td.MakeDigiTwinThingID(agentID, thingID)

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := &td.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agentID, string(tddjson))
	require.NoError(t, err)

	// invoke the action with the wrong thing
	req := messaging.NewRequestMessage(
		wot.OpInvokeAction, "badThingID", actionName, actionValue, correlationID)
	_, stored, err := dtwStore.NewActionStart(req)

	// unknown thingIDs are still allowed for now.
	assert.NoError(t, err)
	assert.False(t, stored)

	// invoke the action with the wrong name
	req = messaging.NewRequestMessage(
		wot.OpInvokeAction, dThingID, "badName", actionValue, correlationID)
	_, stored, err = dtwStore.NewActionStart(req)
	// same as above
	assert.NoError(t, err)
	assert.False(t, stored)

	// complete the action on wrong thing
	resp := messaging.NewResponseMessage(
		wot.OpInvokeAction, "badThingID", actionName, actionValue, nil, correlationID)

	_, err = dtwStore.UpdateActionWithResponse(resp)
	assert.Error(t, err)

	// complete the action on wrong action name
	resp.ThingID = thingID
	resp.Name = "badName"
	_, err = dtwStore.UpdateActionWithResponse(resp)
	assert.Error(t, err)
}

func TestDigitwinAgentAction(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	const agentID = "agent1"
	const thingID = "thing1"
	const title1 = "title1"
	const consumerID = "user1"
	const actionName = "action1"
	const actionValue = 25
	const correlationID = "request-1"

	dThingID := td.MakeDigiTwinThingID(agentID, thingID)

	svc, _, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := &td.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "action 1", "", actionSchema)
	tddJSON1, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateThing(agentID, string(tddJSON1))
	require.NoError(t, err)

	// read back should succeed
	tddJson2, err := svc.DirSvc.RetrieveThing(consumerID, dThingID)
	require.NoError(t, err)
	require.NotEmpty(t, tddJson2)

	// next, invoke the action to read the thing from the directory.
	ag := service.NewDigitwinAgent(svc)
	req := messaging.NewRequestMessage(vocab.OpInvokeAction,
		digitwin.ThingDirectoryDThingID, digitwin.ThingDirectoryRetrieveThingMethod, dThingID, consumerID)
	req.CorrelationID = correlationID
	resp := ag.HandleRequest(req, nil)

	require.Empty(t, resp.Error)
	require.NotEmpty(t, resp.Value)

	// a non-existing TD should fail
	req = messaging.NewRequestMessage(vocab.OpInvokeAction,
		digitwin.ThingDirectoryDThingID, digitwin.ThingDirectoryRetrieveThingMethod, "badid", consumerID)
	req.CorrelationID = correlationID
	resp = ag.HandleRequest(req, nil)
	require.NotEmpty(t, resp.Error)

	// a non-existing method name should fail
	req = messaging.NewRequestMessage(vocab.OpInvokeAction,
		digitwin.ThingDirectoryDThingID, "badMethod", dThingID, consumerID)
	req.CorrelationID = correlationID
	resp = ag.HandleRequest(req, nil)
	require.NotEmpty(t, resp.Error)

	// a non-existing serviceID should fail
	req = messaging.NewRequestMessage(vocab.OpInvokeAction,
		"badservicename", digitwin.ThingDirectoryRetrieveThingMethod, dThingID, consumerID)
	req.CorrelationID = correlationID
	resp = ag.HandleRequest(req, nil)
	require.NotEmpty(t, resp.Error)

}
