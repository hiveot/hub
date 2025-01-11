package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/services/state/stateapi"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// StateAgent agent for the state storage services
type StateAgent struct {
	hc  transports.IAgentConnection
	svc *StateService
}

// CreateTD returns a TD describing the service
func (agent StateAgent) CreateTD() *td.TD {
	tdi := td.NewTD(stateapi.StorageServiceID, "State Store", vocab.ThingService)
	// delete key
	tdi.AddAction(stateapi.DeleteMethod, "Delete State", "Delete state by key",
		&td.DataSchema{
			Title:       "Key",
			Description: "The key whose stored data was added with",
			Type:        wot.WoTDataTypeString,
		})
	// get by key
	a2 := tdi.AddAction(stateapi.GetMethod, "Read State", "Read state data by key",
		&td.DataSchema{
			Title:       "Key",
			Description: "The key whose stored data to read",
			Type:        wot.WoTDataTypeString,
		})
	a2.Safe = true
	a2.Synchronous = true
	a2.Idempotent = true
	a2.Output = &td.DataSchema{
		Title: "Value",
		Type:  wot.WoTDataTypeObject,
		Properties: map[string]*td.DataSchema{
			"key":   {Type: wot.WoTDataTypeString},
			"found": {Type: wot.WoTDataTypeBool},
			"value": {Type: wot.WoTDataTypeString},
		},
	}
	// get multiple
	a3 := tdi.AddAction(stateapi.GetMultipleMethod,
		"Read Multiple States", "",
		&td.DataSchema{
			Title:       "Keys",
			Description: "List of keys whose state to get",
			Type:        wot.WoTDataTypeArray,
		})
	a3.Safe = true
	a3.Synchronous = true
	a3.Idempotent = true
	a3.Output = &td.DataSchema{
		Title:       "Values",
		Description: "Map of state values by key",
		Type:        wot.WoTDataTypeObject,
		Properties: map[string]*td.DataSchema{
			"": {Type: wot.WoTDataTypeObject,
				Properties: map[string]*td.DataSchema{
					"key":   {Type: wot.WoTDataTypeString},
					"found": {Type: wot.WoTDataTypeBool},
					"value": {Type: wot.WoTDataTypeString},
				},
			},
		},
	}
	// set state by key
	a4 := tdi.AddAction(stateapi.SetMethod, "Set State", "Write a state value by key",
		&td.DataSchema{
			Title:       "Key-Value",
			Description: "The key and value to store state data under",
			Type:        wot.WoTDataTypeObject,
			Properties: map[string]*td.DataSchema{
				"key":   {Type: wot.WoTDataTypeString},
				"value": {Type: wot.WoTDataTypeString},
			},
		})
	a4.Safe = false
	a4.Synchronous = true
	a4.Idempotent = true
	// set multiple
	a5 := tdi.AddAction(stateapi.SetMultipleMethod, "Set Multiple", "Write a map of state values",
		&td.DataSchema{
			Title:       "Key-Values",
			Description: "Map with new state values",
			Type:        wot.WoTDataTypeObject,
		})
	a5.Safe = false
	a5.Synchronous = true
	a5.Idempotent = true
	return tdi
}

// HandleRequest dispatches requests to the service capabilities
func (agent *StateAgent) HandleRequest(req transports.RequestMessage) transports.ResponseMessage {
	if req.Operation == wot.OpInvokeAction && req.ThingID == stateapi.StorageServiceID {
		switch req.Name {
		case stateapi.DeleteMethod:
			return agent.Delete(req)
		case stateapi.GetMethod:
			return agent.Get(req)
		case stateapi.GetMultipleMethod:
			return agent.GetMultiple(req)
		case stateapi.SetMethod:
			return agent.Set(req)
		case stateapi.SetMultipleMethod:
			return agent.SetMultiple(req)
		}
	}
	err := fmt.Errorf("unknown action '%s' for service '%s'", req.Name, req.ThingID)
	return req.CreateResponse(nil, err)
}
func (agent *StateAgent) Delete(req transports.RequestMessage) transports.ResponseMessage {
	key := tputils.DecodeAsString(req.Input, 0)
	err := agent.svc.Delete(req.SenderID, key)
	return req.CreateResponse(nil, err)
}
func (agent *StateAgent) Get(req transports.RequestMessage) transports.ResponseMessage {
	var err error
	output := stateapi.GetResp{}
	key := tputils.DecodeAsString(req.Input, 0)
	output.Key = key
	output.Value, output.Found, err = agent.svc.Get(req.SenderID, key)
	return req.CreateResponse(output, err)
}
func (agent *StateAgent) GetMultiple(req transports.RequestMessage) transports.ResponseMessage {
	input := stateapi.GetMultipleArgs{}
	output := stateapi.GetMultipleResp{}
	err := tputils.DecodeAsObject(req.Input, &input)
	if err == nil {
		output, err = agent.svc.GetMultiple(req.SenderID, input)
	}
	return req.CreateResponse(output, err)
}
func (agent *StateAgent) Set(req transports.RequestMessage) transports.ResponseMessage {
	input := stateapi.SetArgs{}
	err := tputils.DecodeAsObject(req.Input, &input)
	if err == nil {
		err = agent.svc.Set(req.SenderID, input.Key, input.Value)
	}
	return req.CreateResponse(nil, err)
}
func (agent *StateAgent) SetMultiple(req transports.RequestMessage) transports.ResponseMessage {
	input := stateapi.SetMultipleArgs{}
	err := tputils.DecodeAsObject(req.Input, &input)
	if err == nil {
		err = agent.svc.SetMultiple(req.SenderID, input)
	}
	return req.CreateResponse(nil, err)
}

// NewStateAgent returns a new instance of the communication agent for the state service.
// Intended for use in StartPlugin
//
//	svc is the state service whose capabilities to expose
func NewStateAgent(svc *StateService) *StateAgent {
	agent := StateAgent{svc: svc}
	return &agent
}

// StartStateAgent returns a new instance of the communication agent for the state service.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
//
//	svc is the state service whose capabilities to expose
//	hc is the messaging client used to register a message handler
func StartStateAgent(svc *StateService, hc transports.IAgentConnection) *StateAgent {
	agent := StateAgent{hc: hc, svc: svc}
	if hc != nil {
		hc.SetRequestHandler(agent.HandleRequest)

		// publish the service TD
		tdi := agent.CreateTD()
		err := hc.PubTD(tdi)
		if err != nil {
			slog.Error("Failed publishing the TD", "err", err.Error())
		}
	}
	return &agent
}
