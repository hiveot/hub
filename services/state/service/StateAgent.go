package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/services/state/stateapi"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// StateAgent agent for the state storage services
type StateAgent struct {
	hc  transports.IClientConnection
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
			Title:       "Key",
			Description: "The key to store state data under",
			Type:        wot.WoTDataTypeString,
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
func (agent *StateAgent) HandleRequest(msg *transports.ThingMessage) (output any, err error) {
	if msg.ThingID == stateapi.StorageServiceID {
		switch msg.Name {
		case stateapi.DeleteMethod:
			return agent.Delete(msg)
		case stateapi.GetMethod:
			return agent.Get(msg)
		case stateapi.GetMultipleMethod:
			return agent.GetMultiple(msg)
		case stateapi.SetMethod:
			return agent.Set(msg)
		case stateapi.SetMultipleMethod:
			return agent.SetMultiple(msg)
		}
	}
	err = fmt.Errorf("unknown action '%s' for service '%s'", msg.Name, msg.ThingID)
	return nil, err
}
func (agent *StateAgent) Delete(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.DeleteArgs{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		err = agent.svc.Delete(msg.SenderID, args.Key)
	}
	return nil, err
}
func (agent *StateAgent) Get(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.GetArgs{}
	resp := stateapi.GetResp{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		resp.Key = args.Key
		resp.Value, resp.Found, err = agent.svc.Get(msg.SenderID, args.Key)
	}
	return resp, err
}
func (agent *StateAgent) GetMultiple(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.GetMultipleArgs{}
	resp := stateapi.GetMultipleResp{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		resp.KV, err = agent.svc.GetMultiple(msg.SenderID, args.Keys)
	}
	return resp, err
}
func (agent *StateAgent) Set(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.SetArgs{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		err = agent.svc.Set(msg.SenderID, args.Key, args.Value)
	}
	return nil, err
}
func (agent *StateAgent) SetMultiple(msg *transports.ThingMessage) (output any, err error) {
	args := stateapi.SetMultipleArgs{}
	err = tputils.DecodeAsObject(msg.Data, &args)
	if err == nil {
		err = agent.svc.SetMultiple(msg.SenderID, args.KV)
	}
	return nil, err
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
func StartStateAgent(svc *StateService, hc transports.IClientConnection) *StateAgent {
	agent := StateAgent{hc: hc, svc: svc}
	if hc != nil {
		hc.SetRequestHandler(agent.HandleRequest)

		// but the service TD
		tdi := agent.CreateTD()
		tdJSON, _ := jsoniter.Marshal(tdi)
		err := hc.SendNotification(wot.HTOpUpdateTD, tdi.ID, "", string(tdJSON))
		if err != nil {
			slog.Error("Failed publishing the TD", "err", err.Error())
		}
	}
	return &agent
}
