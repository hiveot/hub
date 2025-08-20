package service

import (
	"log/slog"

	"github.com/hiveot/hub/messaging"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/runtime/digitwin/store"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
)

// DirectoryService provides the digital twin directory service
// This is based on the W3C WoT Discovery draft specification: https://w3c.github.io/wot-discovery
// Currently being revised to be compatible.
type DirectoryService struct {
	dtwStore *store.DigitwinStore

	// include forms in each affordance when generating the digitwin TD
	includeAffordanceForms bool

	// notifications on directory changes
	notifHandler    messaging.NotificationHandler
	addFormsHandler func(*td.TD, bool)
}

// MakeDigitalTwinTD returns the digital twin from an agent provided TD
// This modifies the TD as follows:
//  1. Change the ThingID to the digitwin thing ID:  dtw:{agentID}:{thingID}
//  2. Change the forms to digitwin supported forms
//  3. Set the securitydefinitions to digitwin supported auth (by forms handler)
//  4. Add a writable 'title' property if it doesn't exist
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
	dtwTD.Security = nil
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
	// the forms handler defines the protocols and security scheme for accessing the digital twin TD
	if svc.addFormsHandler != nil {
		svc.addFormsHandler(dtwTD, svc.includeAffordanceForms)
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
		tdJSON, err = jsoniter.MarshalToString(dtd)
	}
	return tdJSON, err
}

// ReadAllTDs returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DirectoryService) ReadAllTDs(
	senderID string, args digitwin.ThingDirectoryReadAllTDsArgs) (tdList []string, err error) {

	dtdList, err := svc.dtwStore.ReadTDs(args.Offset, args.Limit)
	if err == nil {
		tdList = make([]string, 0, len(dtdList))
		for _, dtd := range dtdList {
			tdJSON, err2 := jsoniter.MarshalToString(dtd)
			if err2 == nil {
				tdList = append(tdList, tdJSON)
			}
		}
	}
	return tdList, err
}

// RemoveTD removes a Thing TD document from the digital twin directory
func (svc *DirectoryService) RemoveTD(senderID string, dThingID string) error {
	err := svc.dtwStore.RemoveDTW(dThingID, senderID)
	if err == nil && svc.notifHandler != nil {
		// Publish an event notifying subscribers that the Thing was removed from the directory
		// Those subscribing to directory event will be notified
		notif := messaging.NewNotificationMessage(wot.OpSubscribeEvent,
			digitwin.ThingDirectoryDThingID, digitwin.ThingDirectoryEventThingRemoved,
			dThingID)
		go svc.notifHandler(notif)
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
	if err != nil {
		return err
	}
	slog.Info("UpdateDTD",
		slog.String("agentID", agentID), slog.String("thingID", thingTD.ID))
	// store both the original and digitwin TD documents
	svc.dtwStore.UpdateTD(agentID, thingTD, digitalTwinTD)

	// notify subscribers of TD updates
	if svc.notifHandler != nil {
		dtdJSON, _ := jsoniter.MarshalToString(digitalTwinTD)
		
		// todo: only send notification on changes
		// publish an event that the directory TD has updated with a new TD
		notif := messaging.NewNotificationMessage(wot.OpSubscribeEvent,
			digitwin.ThingDirectoryDThingID, digitwin.ThingDirectoryEventThingUpdated,
			dtdJSON)
		go svc.notifHandler(notif)
	}
	return err
}

// NewDigitwinDirectoryService creates a new instance of the directory service
// using the given store.
// This is based on the W3C WoT Discovery draft specification: https://w3c.github.io/wot-discovery
// Currently being revised to be compatible.
// The transport binding can be supplied directly or set later by the parent service
func NewDigitwinDirectoryService(
	dtwStore *store.DigitwinStore, notifHandler messaging.NotificationHandler,
	includeAffordanceForms bool) *DirectoryService {

	dirSvc := &DirectoryService{
		dtwStore:               dtwStore,
		notifHandler:           notifHandler,
		includeAffordanceForms: includeAffordanceForms,
	}

	// verify service interface matches the TD generated interface
	var s digitwin.IThingDirectoryService = dirSvc
	_ = s

	return dirSvc
}
