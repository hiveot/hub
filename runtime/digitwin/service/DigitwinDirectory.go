package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// TODO: use constants from TD generated API
const ThingUpdatedEventName = "thingUpdated"
const ThingRemovedEventName = "thingRemoved"

// Digital Twin Directory Service
type DigitwinDirectoryService struct {
	dtwStore *DigitwinStore

	// transport binding for publishing directory events and getting forms
	// to include in digital twin TDs.
	tb api.ITransportBinding
}

// MakeDigitalTwinTD returns the digital twin from a agent provided TD
func (svc *DigitwinDirectoryService) MakeDigitalTwinTD(
	agentID string, tdJSON string) (thingTD *tdd.TD, dtwTD *tdd.TD, err error) {

	err = jsoniter.Unmarshal([]byte(tdJSON), &thingTD)
	if err != nil {
		return thingTD, dtwTD, err
	}
	_ = jsoniter.Unmarshal([]byte(tdJSON), &dtwTD)

	dtwTD.ID = tdd.MakeDigiTwinThingID(agentID, thingTD.ID)

	// remove all existing forms and auth info
	dtwTD.Forms = make([]tdd.Form, 0)
	dtwTD.Security = ""
	dtwTD.SecurityDefinitions = make(map[string]tdd.SecurityScheme)
	for _, aff := range dtwTD.Properties {
		aff.Forms = make([]tdd.Form, 0)
	}
	for _, aff := range dtwTD.Events {
		aff.Forms = make([]tdd.Form, 0)
	}
	for _, aff := range dtwTD.Actions {
		aff.Forms = make([]tdd.Form, 0)
	}

	if svc.tb != nil {
		err = svc.tb.AddTDForms(dtwTD)
	}
	return thingTD, dtwTD, err
}

// QueryDTDs returns a list of JSON encoded TD documents
func (svc *DigitwinDirectoryService) QueryDTDs(
	senderID string, args digitwin.DirectoryQueryDTDsArgs) (tdDocuments []string, err error) {
	//svc.DtwStore.QueryDTDs(args)
	return nil, fmt.Errorf("Not yet implemented")
}

// ReadDTD returns a JSON encoded TD document
func (svc *DigitwinDirectoryService) ReadDTD(senderID string, dThingID string) (tdJSON string, err error) {
	dtd, err := svc.dtwStore.ReadDThing(dThingID)
	if err == nil {
		var tdByte []byte
		tdByte, err = jsoniter.Marshal(dtd)
		tdJSON = string(tdByte)
	}
	return tdJSON, err
}

// ReadAllDTDs returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DigitwinDirectoryService) ReadAllDTDs(
	senderID string, args digitwin.DirectoryReadAllDTDsArgs) (tdList []string, err error) {

	dtdList, err := svc.dtwStore.ReadDTDs(args.Offset, args.Limit)
	if err == nil {
		tdList = make([]string, 0, len(dtdList))
		for _, dtd := range dtdList {
			tdByte, err2 := jsoniter.Marshal(dtd)
			if err2 == nil {
				tdList = append(tdList, string(tdByte))
			}
		}
	}
	return tdList, err
}

// RemoveDTD removes a Thing TD document from the digital twin directory
// FIXME: submit an event (as per TDD) that a TD has been removed
func (svc *DigitwinDirectoryService) RemoveDTD(senderID string, dThingID string) error {
	err := svc.dtwStore.RemoveDTW(dThingID)
	if err == nil && svc.tb != nil {
		// publish the event in the background
		go svc.tb.PublishEvent(
			digitwin.DirectoryDThingID, ThingRemovedEventName, dThingID, "")
	}
	return err
}

// UpdateDTD updates the digitwin TD from an agent supplied TD
// This transforms the given TD into the digital twin instance, stores it in the directory
// store and sends a thing-updated event as described in the TD.
// This returns the updated digital twin TD
// FIXME: submit an event as per TDD that a TD has been updated
func (svc *DigitwinDirectoryService) UpdateDTD(agentID string, tdJson string) error {
	slog.Info("UpdateDTD")

	// transform the agent provided TD into a digital twin's TD
	thingTD, digitalTwinTD, err := svc.MakeDigitalTwinTD(agentID, tdJson)
	svc.dtwStore.UpdateTD(agentID, thingTD, digitalTwinTD)

	dtdJSON, err := jsoniter.Marshal(digitalTwinTD)
	if err == nil && svc.tb != nil {
		// publish the event in the background
		go svc.tb.PublishEvent(
			digitwin.DirectoryDThingID, ThingUpdatedEventName, string(dtdJSON), "")
	}
	return err
}

// NewDigitwinDirectoryService creates a new instance of the directory service
// using the given store.
//
// The transport binding can be supplied directly or set later by the parent service
func NewDigitwinDirectoryService(
	dtwStore *DigitwinStore, tb api.ITransportBinding) *DigitwinDirectoryService {
	dirSvc := &DigitwinDirectoryService{
		dtwStore: dtwStore,
		tb:       tb,
	}

	// verify service interface matches the TD generated interface
	var s digitwin.IDirectoryService = dirSvc
	_ = s

	return dirSvc
}
