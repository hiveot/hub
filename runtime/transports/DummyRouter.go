package transports

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
)

// DummyRouter for implementing test hooks defined in IHubRouter
type DummyRouter struct {
	OnAction func(senderID, dThingID, name string, val any, msgID string, cid string) any
	OnEvent  func(agentID, thingID, name string, val any, msgID string)
}

func (svc *DummyRouter) HandleActionFlow(
	dThingID string, actionName string, input any, reqID string, senderID string, cid string) (
	status string, output any, messageID string, err error) {
	// if a hook is provided, call it first
	if svc.OnAction != nil {
		output = svc.OnAction(senderID, dThingID, actionName, input, messageID, cid)
	}
	return vocab.ProgressStatusDelivered, output, reqID, nil
}

func (svc *DummyRouter) HandleActionProgress(agentID string, stat hubclient.ActionProgress) error {
	return nil
}
func (svc *DummyRouter) HandleEventFlow(
	agentID string, thingID string, name string, value any, messageID string) error {
	if svc.OnEvent != nil {
		svc.OnEvent(agentID, thingID, name, value, messageID)
	}
	return nil
}

func (svc *DummyRouter) HandleUpdatePropertyFlow(
	agentID string, thingID string, propName string, value any, reqID string) error {
	return nil
}
func (svc *DummyRouter) HandleUpdateTDFlow(agentID string, tdJSON string) error {
	return nil
}

func (svc *DummyRouter) HandleWritePropertyFlow(
	dThingID string, name string, newValue any, consumerID string) (
	status string, messageID string, err error) {

	return "", "", nil
}
