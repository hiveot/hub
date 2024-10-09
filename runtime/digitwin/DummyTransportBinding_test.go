package digitwin_test

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/digitwin"
	"github.com/hiveot/hub/wot/tdd"
)

// dummy transport for testing with the digitwin service
// This implements the ITransportBinding interface.
type DummyTransportBinding struct {
}

func (dummy *DummyTransportBinding) AddTDForms(td *tdd.TD) error {
	return nil
}
func (dummy *DummyTransportBinding) GetProtocolInfo() api.ProtocolInfo {
	return api.ProtocolInfo{}
}

func (dummy *DummyTransportBinding) InvokeAction(agentID string, thingID string, name string, value any, messageID string) (
	status string, output any, err error) {
	return digitwin.StatusPending, nil, nil
}

func (dummy *DummyTransportBinding) PublishEvent(dThingID string, name string, value any, messageID string) {
}

func (dummy *DummyTransportBinding) PublishProperty(dThingID string, name string, value any, messageID string) {
}
func (dummy *DummyTransportBinding) PublishActionProgress(connectionID string, stat hubclient.DeliveryStatus) error {
	return nil
}

func (dummy *DummyTransportBinding) WriteProperty(agentID string, thingID string, name string, value any, msgID string) (status string, err error) {
	return digitwin.StatusPending, nil
}
func NewDummyTransportBinding() api.ITransportBinding {
	dummy := DummyTransportBinding{}
	return &dummy
}
