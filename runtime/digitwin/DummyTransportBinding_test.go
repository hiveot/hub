package digitwin_test

import (
	"github.com/hiveot/hub/api/go/vocab"
	transports2 "github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot/td"
)

// dummy transport for testing with the digitwin service
// This implements the ITransportBinding interface.
type DummyTransportBinding struct {
}

func (dummy *DummyTransportBinding) AddTDForms(td *td.TD) error {
	return nil
}
func (dummy *DummyTransportBinding) GetConnectionByConnectionID(cid string) transports2.IServerConnection {
	return nil
}
func (dummy *DummyTransportBinding) GetProtocolType() string {
	return transports2.ProtocolTypeEmbedded
}

func (dummy *DummyTransportBinding) InvokeAction(
	agentID string, thingID string, name string, value any, requestID string, consumerID string) (
	status string, output any, err error) {

	return vocab.RequestPending, nil, nil
}

func (dummy *DummyTransportBinding) PublishEvent(
	dThingID string, name string, value any, requestID string, agentID string) {
}

func (dummy *DummyTransportBinding) PublishProperty(
	dThingID string, name string, value any, requestID string, agentID string) {
}
func (dummy *DummyTransportBinding) PublishProgressUpdate(
	connectionID string, stat transports2.RequestStatus, agentID string) (bool, error) {
	return false, nil
}

func (dummy *DummyTransportBinding) WriteProperty(
	agentID string, thingID string, name string, value any, msgID string, senderID string) (
	found bool, status string, err error) {

	return false, vocab.RequestPending, nil
}

func NewDummyTransportBinding() *DummyTransportBinding {
	dummy := DummyTransportBinding{}
	return &dummy
}
