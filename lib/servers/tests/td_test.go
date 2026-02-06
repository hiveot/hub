package tests

import (
	"testing"

	"github.com/hiveot/hivekit/go/wot/td"
)

// test TD messages and forms
// this uses the client and server helpers defined in connect_test.go

const DeviceTypeSensor = "hiveot:sensor"

//// Test consumer reads a TD from agent via the server
//func TestReadTDFromAgent(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//	var thingID = "thing1"
//
//	// 1. start the transport
//	srv, cancelFn := StartTransportServer(nil, nil)
//	_ = srv
//	defer cancelFn()
//
//	// 2. connect as an agent
//	agConn1, ag1, _ := NewAgent(testAgentID1)
//	defer agConn1.Disconnect()
//
//	// 3. agent creates TD
//	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)
//
//	// agent request handler to read TD
//	agentReqHandler := func(req *transports.RequestMessage,
//		connection transports.IConnection) *transports.ResponseMessage {
//		t.Log("Received request: " + req.Operation)
//		if req.Operation == wot.HTOpReadTD {
//			tdJSON, err := jsoniter.Marshal(td1)
//			return req.CreateResponse(tdJSON, err)
//		} else if req.Operation == wot.HTOpReadAllTDs {
//			tdJSON, err := jsoniter.Marshal(td1)
//			return req.CreateResponse([]string{string(tdJSON)}, err)
//		} else {
//			return req.CreateResponse(nil,
//				errors.New("agent receives unknown request: "+req.Operation))
//		}
//	}
//	ag1.SetRequestHandler(agentReqHandler)
//
//	// 4. verify the TD can be read from the agent
//	c := srv.GetConnectionByClientID(testAgentID1)
//	require.True(t, c.IsConnected())
//	// c is server side connection of the agent. The hub is the consumer of the agent.
//	consumer := consumer.NewConsumer(c, testTimeout)
//	tdList, err := consumer.ReadAllTDs()
//	require.NoError(t, err)
//	require.True(t, len(tdList) > 0)
//
//	var td2 td.TD
//	err = jsoniter.UnmarshalFromString(tdList[0], &td2)
//	require.NoError(t, err)
//	assert.Equal(t, td1.ID, td2.ID)
//	assert.Equal(t, td1.Title, td2.Title)
//	assert.Equal(t, td1.AtType, td2.AtType)
//}

// Test if forms are indeed added to a TD, describing the transport protocol binding operations
func TestAddForms(t *testing.T) {
	t.Logf("---%s---\n", t.Name())
	var thingID = "thing1"

	// handler of TDs on the server
	// 1. start the transport
	_, cancelFn := StartTransportServer(nil, nil, nil)
	defer cancelFn()

	// 2. Create a TD
	tdi := td.NewTD(thingID, "My gadget", DeviceTypeSensor)

	// 3. add forms
	transportServer.AddTDForms(tdi, true)

	// 4. Check that at least 1 form are present
	// TODO: add the hiveot endpoints
	//assert.GreaterOrEqual(t, len(tdi.Forms), 1)
}

//// Agent Publishes TD to the directory
//func TestPublishTD(t *testing.T) {
//	t.Logf("---%s---\n", t.Name())
//	var thingID = "thing1"
//	var rxTD atomic.Value
//
//	// 2. Create a TD
//	td1 := td.NewTD(thingID, "My gadget", DeviceTypeSensor)
//	td1JSON, _ := jsoniter.MarshalToString(td1)
//
//	// handler of TDs on the server
//	requestHandler := func(msg *transports.RequestMessage,
//		c transports.IConnection) *transports.ResponseMessage {
//		var err error
//		if msg.Operation == wot.HTOpUpdateTD {
//			assert.Equal(t, thingID, msg.ThingID)
//			assert.Equal(t, td1JSON, msg.Input)
//			assert.NotEmpty(t, msg.Input)
//			rxTD.Store(msg.Input)
//		} else {
//			err = fmt.Errorf("Unexpected operation: %s" + msg.Operation)
//		}
//		resp := msg.CreateResponse(nil, err)
//		return resp
//	}
//
//	// 1. start the transport server with the TD handler
//	srv, cancelFn := StartTransportServer(requestHandler, nil)
//	_ = srv
//	defer cancelFn()
//
//	// 2. Connect as agent
//	cc1, ag1, _ := NewAgent(testAgentID1)
//	defer cc1.Disconnect()
//
//	// Agent publishes the TD
//	err := ag1.UpdateThing(td1)
//	require.NoError(t, err)
//	time.Sleep(time.Millisecond * 10)
//
//	// check reception
//	require.Equal(t, td1JSON, rxTD.Load())
//}
