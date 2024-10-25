package digitwin_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/wot/tdd"
)

// dummy transport for testing with the digitwin service
// This implements the ITransportBinding interface.
type DummyTransportBinding struct {
}

func (dummy *DummyTransportBinding) AddTDForms(td *tdd.TD) error {
	return nil
}
func (dummy *DummyTransportBinding) GetConnectionByCID(cid string) connections.IClientConnection {
	return nil
}
func (dummy *DummyTransportBinding) GetProtocolInfo() api.ProtocolInfo {
	return api.ProtocolInfo{}
}

func (dummy *DummyTransportBinding) InvokeAction(
	agentID string, thingID string, name string, value any, messageID string, consumerID string) (
	status string, output any, err error) {

	return vocab.ProgressStatusPending, nil, nil
}

func (dummy *DummyTransportBinding) PublishEvent(
	dThingID string, name string, value any, messageID string, agentID string) {
}

func (dummy *DummyTransportBinding) PublishProperty(
	dThingID string, name string, value any, messageID string, agentID string) {
}
func (dummy *DummyTransportBinding) PublishProgressUpdate(
	connectionID string, stat hubclient.ActionProgress, agentID string) (bool, error) {
	return false, nil
}

func (dummy *DummyTransportBinding) WriteProperty(
	agentID string, thingID string, name string, value any, msgID string, senderID string) (
	found bool, status string, err error) {

	return false, vocab.ProgressStatusPending, nil
}

func NewDummyTransportBinding() api.ITransportBinding {
	dummy := DummyTransportBinding{}
	return &dummy
}
