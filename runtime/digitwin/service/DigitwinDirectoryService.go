package service

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/runtime/connections"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// Digital Twin Directory Service
type DigitwinDirectoryService struct {
	dtwStore *DigitwinStore

	// transport binding for publishing directory events and getting forms
	// to include in digital twin TDs.
	//tb api.ITransportBinding
	cm              *connections.ConnectionManager
	addFormsHandler func(*tdd.TD) error
}

// MakeDigitalTwinTD returns the digital twin from a agent provided TD
func (svc *DigitwinDirectoryService) MakeDigitalTwinTD(
	agentID string, tdJSON string) (thingTD *tdd.TD, dtwTD *tdd.TD, err error) {

	err = jsoniter.UnmarshalFromString(tdJSON, &thingTD)
	if err != nil {
		slog.Error("MakeDigitalTwinTD. Bad TD", "err", err.Error())
		return thingTD, dtwTD, err
	}
	// make a deep copy for the digital twin
	_ = jsoniter.UnmarshalFromString(tdJSON, &dtwTD)

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

	if svc.addFormsHandler != nil {
		err = svc.addFormsHandler(dtwTD)
	}
	return thingTD, dtwTD, err
}

//func (svc *DigitwinDirectoryService) QueryDTDs(
//	senderID string, args digitwin.DirectoryQueryTDsArgs) (tdDocuments []string, err error) {
//	//svc.DtwStore.QueryDTDs(args)
//	return nil, fmt.Errorf("Not yet implemented")
//}

// ReadTD returns a JSON encoded TD document
func (svc *DigitwinDirectoryService) ReadTD(senderID string, dThingID string) (tdJSON string, err error) {
	dtd, err := svc.dtwStore.ReadDThing(dThingID)
	if err == nil {
		var tdByte []byte
		tdByte, err = jsoniter.Marshal(dtd)
		tdJSON = string(tdByte)
	}
	return tdJSON, err
}

// ReadAllTDs returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DigitwinDirectoryService) ReadAllTDs(
	senderID string, args digitwin.DirectoryReadAllTDsArgs) (tdList []string, err error) {

	dtdList, err := svc.dtwStore.ReadTDs(args.Offset, args.Limit)
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

// RemoveTD removes a Thing TD document from the digital twin directory
func (svc *DigitwinDirectoryService) RemoveTD(senderID string, dThingID string) error {
	err := svc.dtwStore.RemoveDTW(dThingID, senderID)
	if err == nil && svc.cm != nil {
		// notify subscribers that the digital twin thing was removed.
		// payload is the digital twin's ID
		go svc.cm.PublishEvent(
			digitwin.DirectoryDThingID, digitwin.DirectoryEventThingRemoved, dThingID,
			"", digitwin.DirectoryAgentID)
	}
	return err
}

// UpdateTD updates the digitwin TD from an agent supplied TD
// This transforms the given TD into the digital twin instance, stores it in the directory
// store and sends a thing-updated event as described in the TD.
// This returns the updated digital twin TD
// FIXME: only agents are allowed this
func (svc *DigitwinDirectoryService) UpdateTD(agentID string, tdJson string) error {

	// transform the agent provided TD into a digital twin's TD
	thingTD, digitalTwinTD, err := svc.MakeDigitalTwinTD(agentID, tdJson)
	slog.Info("UpdateDTD",
		slog.String("agentID", agentID), slog.String("thingID", thingTD.ID))
	// store both the original and digitwin TD documents
	svc.dtwStore.UpdateTD(agentID, thingTD, digitalTwinTD)

	// notify subscribers of TD updates
	if err == nil && svc.cm != nil {
		dtdJSON, _ := jsoniter.Marshal(digitalTwinTD)
		// FIXME: notify subscribers that the TD has been updated
		//svc.cm.PublishTD()
		//ForEachConnection

		go svc.cm.PublishEvent(
			digitwin.DirectoryDThingID, digitwin.DirectoryEventThingUpdated, string(dtdJSON),
			"", digitwin.DirectoryAgentID)
	}
	return err
}

// NewDigitwinDirectoryService creates a new instance of the directory service
// using the given store.
//
// The transport binding can be supplied directly or set later by the parent service
func NewDigitwinDirectoryService(
	dtwStore *DigitwinStore, cm *connections.ConnectionManager) *DigitwinDirectoryService {

	dirSvc := &DigitwinDirectoryService{
		dtwStore: dtwStore,
		cm:       cm,
	}

	// verify service interface matches the TD generated interface
	var s digitwin.IDirectoryService = dirSvc
	_ = s

	return dirSvc
}
