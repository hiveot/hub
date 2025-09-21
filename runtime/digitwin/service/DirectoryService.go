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

// CreateThing uses updateThing
func (svc *DirectoryService) CreateThing(agentID string, tdJson string) error {
	return svc.UpdateThing(agentID, tdJson)
}

// DeleteThing removes a Thing TD document from the digital twin directory
func (svc *DirectoryService) DeleteThing(senderID string, dThingID string) error {
	err := svc.dtwStore.RemoveDTW(dThingID, senderID)
	if err == nil && svc.notifHandler != nil {
		// Publish an event notifying subscribers that the Thing was removed from the directory
		// Those subscribing to directory event will be notified
		notif := messaging.NewNotificationMessage(wot.OpSubscribeEvent,
			digitwin.ThingDirectoryDThingID, digitwin.ThingDirectoryEventThingDeleted,
			dThingID)
		go svc.notifHandler(notif)
	}
	return err
}

// MakeDigitalTwinTD returns the digital twin from an agent provided TD
// This modifies the TD as follows:
//  1. Make a deep copy of the original TD
//  2. Change the ThingID to the digitwin thing ID:  dtw:{agentID}:{thingID}
//  3. Reset forms and security definitions
//  4. Use the AddForms hook to populate the digitin TD instance with the forms and security
//     definitions available through the enabled protocols.
//  5. Add a writable 'title' property if it doesn't exist
func (svc *DirectoryService) MakeDigitalTwinTD(
	agentID string, tdJSON string) (thingTD *td.TD, dtwTD *td.TD, err error) {

	err = jsoniter.UnmarshalFromString(tdJSON, &thingTD)
	if err != nil {
		slog.Error("MakeDigitalTwinTD. Bad TD", "err", err.Error())
		return thingTD, dtwTD, err
	}
	// 1. make a deep copy for the digital twin
	_ = jsoniter.UnmarshalFromString(tdJSON, &dtwTD)

	// 2. Change the ThingID to the digital twins ID by prefixing the agent ID
	dtwTD.ID = td.MakeDigiTwinThingID(agentID, thingTD.ID)

	// 3. reset all existing forms and auth info
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

	// 4. populate the TD with forms and security definitions of the available protocols
	// See messaging.servers.TransportManager.AddTDForms() for more details
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

// RetrieveThing returns a JSON encoded TD document
func (svc *DirectoryService) RetrieveThing(senderID string, dThingID string) (tdJSON string, err error) {
	dtd, err := svc.dtwStore.ReadDThing(dThingID)
	if err == nil {
		tdJSON, err = jsoniter.MarshalToString(dtd)
	}
	return tdJSON, err
}

// RetrieveAllThings returns a batch of TD documents
// This returns a list of JSON encoded digital twin TD documents
func (svc *DirectoryService) RetrieveAllThings(
	senderID string, args digitwin.ThingDirectoryRetrieveAllThingsArgs) (tdList []string, err error) {

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

// UpdateThing updates the digitwin TD from an agent supplied TD
// This transforms the given TD into the digital twin instance, stores it in the directory
// store and sends a thing-updated event as described in the TD.
// This returns true when the TD has changed or an error
func (svc *DirectoryService) UpdateThing(agentID string, tdJson string) error {

	// Transform the original TD into a digital twin's TD. This replaces forms and
	// adds the security info for accessing the digital twin Thing.
	thingTD, digitalTwinTD, err := svc.MakeDigitalTwinTD(agentID, tdJson)
	if err != nil {
		return err
	}
	slog.Info("UpdateDTD",
		slog.String("agentID", agentID), slog.String("thingID", thingTD.ID))
	// Store both the original and digitwin TD documents
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
