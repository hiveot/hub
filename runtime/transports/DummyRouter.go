package transports

import "github.com/hiveot/hub/lib/hubclient"

// DummyRouter for implementing test hooks defined in IHubRouter
type DummyRouter struct {
	OnAction func(senderID, dThingID, name string, val any, msgID string) any
	OnEvent  func(agentID, thingID, name string, val any, msgID string)
}

func (svc *DummyRouter) HandleActionFlow(
	consumerID string, dThingID string, actionName string, input any, reqID string) (
	status string, output any, messageID string, err error) {
	// if a hook is provided, call it first
	if svc.OnAction != nil {
		output = svc.OnAction(consumerID, dThingID, actionName, input, messageID)
	}
	return hubclient.DeliveryDelivered, output, reqID, nil
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

func (svc *DummyRouter) HandleWritePropertyFlow(
	consumerID string, dThingID string, name string, newValue any) (
	status string, messageID string, err error) {

	return "", "", nil
}
