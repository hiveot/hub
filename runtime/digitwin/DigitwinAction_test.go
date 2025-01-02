package digitwin_test

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
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
	const correlationID = "req-1"
	dThingID := td.MakeDigiTwinThingID(agentID, thingID)

	svc, dtwStore, stopFunc := startService(true)
	defer stopFunc()

	// Create the native TD for invoking an action to
	tdDoc1 := createTDDoc(thingID, 5, 4, 3)
	actionSchema := &td.DataSchema{Type: vocab.WoTDataTypeInteger, Title: "Position"}
	tdDoc1.AddAction(actionName, "action 1", "", actionSchema)
	tddjson, _ := json.Marshal(tdDoc1)
	err := svc.DirSvc.UpdateTD(agentID, string(tddjson))
	require.NoError(t, err)

	// update the action
	req := transports.NewRequestMessage(
		wot.OpInvokeAction, dThingID, actionName, actionValue, correlationID)
	stored, err := dtwStore.NewActionStart(req)
	require.NoError(t, err)
	require.True(t, stored)

	// check progress
	as, err := svc.ValuesSvc.QueryAction(consumerID, digitwin.ValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    actionName})
	require.NoError(t, err)
	inputVal := tputils.DecodeAsInt(as.Input)
	require.Equal(t, actionValue, inputVal)
	require.Equal(t, correlationID, as.CorrelationID)

	// complete the action
	resp := transports.NewResponseMessage(
		wot.OpInvokeAction, dThingID, actionName, actionValue, nil, correlationID)
	as, err = dtwStore.UpdateActionStatus(agentID, resp)
	require.NoError(t, err)
	require.Equal(t, correlationID, as.CorrelationID)

	// read action status
	as, err = svc.ValuesSvc.QueryAction(consumerID, digitwin.ValuesQueryActionArgs{
		ThingID: dThingID,
		Name:    actionName})

	require.NoError(t, err)
	outputInt := tputils.DecodeAsInt(as.Output)
	require.Equal(t, actionValue, outputInt)
	require.Equal(t, vocab.RequestCompleted, as.Status)

	// read all actions
	//actList, err := svc.ValuesSvc.QueryAllActions(consumerID, dThingID)
	//require.NoError(t, err)
	//require.NotZero(t, len(actList))
}

func TestActionReadFail(t *testing.T) {
	const agentID = "agent1"
	const thingID = "thing1"
	var dThingID = td.MakeDigiTwinThingID(agentID, thingID)

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
	//_, err = svc.ValuesSvc.QueryAllActions("itsme", "badthingid")
	//assert.Error(t, err)
}

func TestInvokeActionErrors(t *testing.T) {
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
	err := svc.DirSvc.UpdateTD(agentID, string(tddjson))
	require.NoError(t, err)

	// invoke the action with the wrong thing
	req := transports.NewRequestMessage(
		wot.OpInvokeAction, "badThingID", actionName, actionValue, correlationID)
	stored, err := dtwStore.NewActionStart(req)

	// unknown thingIDs are still allowed for now.
	assert.NoError(t, err)
	assert.False(t, stored)

	// invoke the action with the wrong name
	req = transports.NewRequestMessage(
		wot.OpInvokeAction, dThingID, "badName", actionValue, correlationID)
	stored, err = dtwStore.NewActionStart(req)
	// same as above
	assert.NoError(t, err)
	assert.False(t, stored)

	// complete the action on wrong thing
	resp := transports.NewResponseMessage(
		wot.OpInvokeAction, "badThingID", actionName, actionValue, nil, correlationID)
	resp.Status = transports.StatusPending
	_, err = dtwStore.UpdateActionStatus(agentID, resp)
	assert.Error(t, err)

	// complete the action on wrong action name
	resp.ThingID = thingID
	resp.Name = "badName"
	_, err = dtwStore.UpdateActionStatus(agentID, resp)
	assert.Error(t, err)
}

func TestDigitwinAgentAction(t *testing.T) {
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
	err := svc.DirSvc.UpdateTD(agentID, string(tddJSON1))
	require.NoError(t, err)

	// read back should succeed
	tddJson2, err := svc.DirSvc.ReadTD(consumerID, dThingID)
	require.NoError(t, err)
	require.NotEmpty(t, tddJson2)

	// next, invoke the action to read the thing from the directory.
	ag := service.NewDigitwinAgent(svc)
	req := transports.NewRequestMessage(vocab.OpInvokeAction,
		digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, dThingID, consumerID)
	req.CorrelationID = correlationID
	resp := ag.HandleRequest(req)
	require.Empty(t, resp.Error)
	require.NotEmpty(t, resp.Output)

	// a non-existing TD should fail
	req = transports.NewRequestMessage(vocab.OpInvokeAction,
		digitwin.DirectoryDThingID, digitwin.DirectoryReadTDMethod, "badid", consumerID)
	req.CorrelationID = correlationID
	resp = ag.HandleRequest(req)
	require.NotEmpty(t, resp.Error)

	// a non-existing method name should fail
	req = transports.NewRequestMessage(vocab.OpInvokeAction,
		digitwin.DirectoryDThingID, "badMethod", dThingID, consumerID)
	req.CorrelationID = correlationID
	resp = ag.HandleRequest(req)
	require.NotEmpty(t, resp.Error)

	// a non-existing serviceID should fail
	req = transports.NewRequestMessage(vocab.OpInvokeAction,
		"badservicename", digitwin.DirectoryReadTDMethod, dThingID, consumerID)
	req.CorrelationID = correlationID
	resp = ag.HandleRequest(req)
	require.NotEmpty(t, resp.Error)

}
