package service

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/connections"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// Digital Twin Directory Service
type DirectoryService struct {
	dtwStore *store.DigitwinStore

	// transport binding for publishing directory events and getting forms
	// to include in digital twin TDs.
	//tb api.ITransportBinding
	cm              *connections.ConnectionManager
	addFormsHandler func(*td.TD) error
}

// MakeDigitalTwinTD returns the digital twin from a agent provided TD
func (svc *DirectoryService) MakeDigitalTwinTD(
	agentID string, tdJSON string) (thingTD *td.TD, dtwTD *td.TD, err error) {

	err = jsoniter.UnmarshalFromString(tdJSON, &thingTD)
	if err != nil {
		slog.Error("MakeDigitalTwinTD. Bad TD", "err", err.Error())
		return thingTD, dtwTD, err
	}
	// make a deep copy for the digital twin
	_ = jsoniter.UnmarshalFromString(tdJSON, &dtwTD)

	dtwTD.ID = td.MakeDigiTwinThingID(agentID, thingTD.ID)

	// remove all existing forms and auth info
	dtwTD.Forms = make([]td.Form, 0)
	dtwTD.Security = ""
	dtwTD.SecurityDefinitions = make(map[string]td.SecurityScheme)
	for _, aff := range dtwTD.Properties {
		aff.Forms = make([]td.Form, 0)
	}
	for _, aff := range dtwTD.Events {
		aff.Forms = make([]td.Form, 0)
	}
	for _, aff := range dtwTD.Actions {
		aff.Forms = make([]td.Form, 0)
	}

	if svc.addFormsHandler != nil {
		err = svc.addFormsHandler(dtwTD)
	}
	return thingTD, dtwTD, err
}

//func (svc *DirectoryService) QueryDTDs(
//	senderID string, args digitwin.DirectoryQueryTDsArgs) (tdDocuments []string, err error) {
//	//svc.DtwStore.QueryDTDs(args)
//	return nil, fmt.Errorf("Not yet implemented")
//}

// ReadTD returns a JSON encoded TD document
func (svc *DirectoryService) ReadTD(senderID string, dThingID string) (tdJSON string, err error) {
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
func (svc *DirectoryService) ReadAllTDs(
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
func (svc *DirectoryService) RemoveTD(senderID string, dThingID string) error {
	err := svc.dtwStore.RemoveDTW(dThingID, senderID)
	if err == nil && svc.cm != nil {
		// Publish an event notifying subscribers that the Thing was removed from the directory
		// Those subscribing to directory event will be notified
		notif := transports.NewNotificationMessage(
			wot.HTOpEvent, digitwin.DirectoryDThingID, digitwin.DirectoryEventThingRemoved, dThingID)
		go svc.cm.PublishNotification(notif)
	}
	return err
}

// UpdateTD updates the digitwin TD from an agent supplied TD
// This transforms the given TD into the digital twin instance, stores it in the directory
// store and sends a thing-updated event as described in the TD.
// This returns true when the TD has changed or an error
func (svc *DirectoryService) UpdateTD(agentID string, tdJson string) error {

	// transform the agent provided TD into a digital twin's TD
	thingTD, digitalTwinTD, err := svc.MakeDigitalTwinTD(agentID, tdJson)
	slog.Info("UpdateDTD",
		slog.String("agentID", agentID), slog.String("thingID", thingTD.ID))
	if err != nil {
		return err
	}
	// store both the original and digitwin TD documents
	svc.dtwStore.UpdateTD(agentID, thingTD, digitalTwinTD)

	// notify subscribers of TD updates
	if svc.cm != nil {
		dtdJSON, _ := jsoniter.Marshal(digitalTwinTD)
		// todo: only send notification on changes
		// publish an event that the directory TD has updated with a new TD
		notif := transports.NewNotificationMessage(
			wot.HTOpEvent, digitwin.DirectoryDThingID, digitwin.DirectoryEventThingUpdated, string(dtdJSON))
		go svc.cm.PublishNotification(notif)
	}
	return err
}

// NewDigitwinDirectoryService creates a new instance of the directory service
// using the given store.
//
// The transport binding can be supplied directly or set later by the parent service
func NewDigitwinDirectoryService(
	dtwStore *store.DigitwinStore, cm *connections.ConnectionManager) *DirectoryService {

	dirSvc := &DirectoryService{
		dtwStore: dtwStore,
		cm:       cm,
	}

	// verify service interface matches the TD generated interface
	var s digitwin.IDirectoryService = dirSvc
	_ = s

	return dirSvc
}