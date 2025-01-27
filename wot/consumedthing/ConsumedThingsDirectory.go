package consumedthing

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/messaging"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"sync"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
const ReadDirLimit = 1000

// ConsumedThingsDirectory manages the consumed things of a servient
//
// This maintains a single instance of each ConsumedThing and updates it when
// an event and action progress updates are received.
type ConsumedThingsDirectory struct {
	consumer *messaging.Consumer
	// Things used by the client
	consumedThings map[string]*ConsumedThing
	// directory of TD documents
	directory map[string]*td.TD
	// the full directory has been read in this session
	fullDirectoryRead bool
	// non-request responses to notify with popup
	responseHandler func(msg *transports.ResponseMessage)
	mux             sync.RWMutex
}

// Consume creates a ConsumedThing instance for reading and operating a Thing.
//
// If an instance already exists it is returned, otherwise one is created using
// the ThingDescription. If the TD is unknown then request it from the directory
// This returns an error if a valid Thing cannot be found.
func (cts *ConsumedThingsDirectory) Consume(thingID string) (ct *ConsumedThing, err error) {
	cts.mux.RLock()
	ct, found := cts.consumedThings[thingID]
	cts.mux.RUnlock()
	if !found || true {
		// if the TD is known then use it
		cts.mux.RLock()
		td, found2 := cts.directory[thingID]
		cts.mux.RUnlock()
		if !found2 {
			// request the TD from the Hub
			td, err = cts.ReadTD(thingID)
			if err != nil {
				return nil, err
			}
		}
		ct = NewConsumedThing(td, cts.consumer)
		ct.ReadAllEvents()
		ct.ReadAllProperties()
		ct.ReadAllActions()
		cts.mux.Lock()
		cts.consumedThings[thingID] = ct
		cts.mux.Unlock()
	}
	return ct, nil
}

// GetForm returns the form for an operation on a Thing
func (cts *ConsumedThingsDirectory) GetForm(op, thingID, name string) (f td.Form) {
	tdi := cts.GetTD(thingID)
	if tdi != nil {
		f = tdi.GetForm(op, name, "")
	}
	return f
}

// IsActive returns whether the session has a connection to the Hub or is in the process of connecting.
//func (cs *ConsumedThingsDirectory) IsActive() bool {
//	status := cs.consumer.GetStatus()
//	return status.ConnectionStatus == hubclient.Connected ||
//		status.ConnectionStatus == hubclient.Connecting
//}

// GetTD returns the TD in the session or nil if thingID isn't found
func (cts *ConsumedThingsDirectory) GetTD(thingID string) *td.TD {
	cts.mux.RLock()
	defer cts.mux.RUnlock()
	// FIXME: get the TD from the server if it isn't in the cache
	tdi := cts.directory[thingID]
	return tdi
}

// handleNotification updates the consumed things from subscriptions
func (cts *ConsumedThingsDirectory) handleNotification(msg transports.ResponseMessage) {

	slog.Debug("CTS.handleNotification",
		slog.String("senderID", msg.SenderID),
		slog.String("operation", msg.Operation),
		slog.String("thingID", msg.ThingID),
		slog.String("name", msg.Name),
		slog.String("clientID (me)", cts.consumer.GetClientID()),
	)

	// if an event is received from an unknown Thing then (re)load its TD
	// progress updates don't count
	// FIXME: delivery updates are no longer notifications
	//if msg.Operation != wot.HTOpUpdateActionStatus {
	//	cts.mux.RLock()
	//	_, found := cts.directory[msg.ThingID]
	//	cts.mux.RUnlock()
	//	if !found {
	//		// FIXME: the digitwin directory TD should be readable. It isnt.
	//		_, err := cts.ReadTD(msg.ThingID)
	//		if err != nil {
	//			slog.Error("Received message with thingID that doesn't exist",
	//				"operation", msg.Operation, "correlationID", msg.CorrelationID,
	//				"thingID", msg.ThingID, "name", msg.Name, "senderID", msg.SenderID)
	//		}
	//	}
	//}
	// update the TD of consumed things
	// the directory service publishes TD updates as events
	// FIXME: use the WoT discovery/directory definition instead of using a digitwin dependency
	// alt: add a PubTD operation
	if msg.Operation == wot.OpSubscribeEvent &&
		msg.ThingID == digitwin.DirectoryDThingID &&
		msg.Name == digitwin.DirectoryEventThingUpdated {
		// decode the TD
		tdi := &td.TD{}
		err := jsoniter.UnmarshalFromString(msg.ToString(0), &tdi)
		if err != nil {
			slog.Error("invalid payload for TD event. Ignored",
				"thingID", msg.ThingID)
			return
		}
		// update consumed thing, if existing
		cts.mux.Lock()
		cts.directory[tdi.ID] = tdi
		ct, found := cts.consumedThings[tdi.ID]
		cts.mux.Unlock()
		if found {
			ct.OnTDUpdate(tdi)
		}
	} else if msg.Operation == wot.OpObserveProperty {
		// update consumed thing, if existing
		cts.mux.Lock()
		defer cts.mux.Unlock()
		ct, found := cts.consumedThings[msg.ThingID]
		if found {
			ct.OnPropertyUpdate(msg)
		}

	} else {
		// this is a regular value event
		cts.mux.RLock()
		ct, found := cts.consumedThings[msg.ThingID]
		cts.mux.RUnlock()
		if found {
			ct.OnEvent(msg)
		}
	}
	// pass it on to chained handler
	if cts.responseHandler != nil {
		cts.responseHandler(&msg)
	}
}

// handleAsyncResponse handles async response of long running actions
func (cts *ConsumedThingsDirectory) handleAsyncResponse(msg *transports.ResponseMessage) error {
	cts.mux.RLock()
	ct, found := cts.consumedThings[msg.ThingID]
	cts.mux.RUnlock()
	if found {
		ct.OnAsyncResponse(msg)
	}

	// pass it on to chained handler
	if cts.responseHandler != nil {
		cts.responseHandler(msg)
	}
	return nil
}

// ReadDirectory loads and decodes Thing Description documents from the directory.
// These are used by Consume()
// This currently limits the nr of things to ReadDirLimit.
//
// Set force to force a reload of the directory instead of using any cached TD documents.
// TODO: reload when things were added or removed
func (cts *ConsumedThingsDirectory) ReadDirectory(force bool) (map[string]*td.TD, error) {
	cts.mux.RLock()
	fullDirectoryRead := cts.fullDirectoryRead
	currentDir := cts.directory
	cts.mux.RUnlock()

	// return the cached version
	if !force && fullDirectoryRead {
		return currentDir, nil
	}
	newDir := make(map[string]*td.TD)

	// TODO: support for reading in pages
	// TODO: use a WoT discovery/directory API instead of a digitwin api
	thingsList, err := digitwin.DirectoryReadAllTDs(cts.consumer, ReadDirLimit, 0)
	if err != nil {
		return newDir, err
	}
	for _, tdJson := range thingsList {
		tdi := td.TD{}
		err = jsoniter.UnmarshalFromString(tdJson, &tdi)
		if err == nil {
			newDir[tdi.ID] = &tdi
		}
	}
	cts.mux.Lock()
	defer cts.mux.Unlock()
	cts.fullDirectoryRead = true
	cts.directory = newDir
	return newDir, nil
}

// ReadTD reads a TD from the directory service and updates the directory cache.
func (cts *ConsumedThingsDirectory) ReadTD(thingID string) (*td.TD, error) {
	// request the TD from the Hub
	tdi := &td.TD{}
	tdJson, err := digitwin.DirectoryReadTD(cts.consumer, thingID)
	if err == nil {
		err = jsoniter.UnmarshalFromString(tdJson, &tdi)
	}
	if err != nil {
		return nil, err
	}
	cts.mux.Lock()
	defer cts.mux.Unlock()
	cts.directory[thingID] = tdi
	return tdi, err
}

// SetResponseHandler registers a handler to receive async responses including events and property updates
func (cts *ConsumedThingsDirectory) SetResponseHandler(handler func(message *transports.ResponseMessage)) {
	cts.mux.Lock()
	defer cts.mux.Unlock()
	cts.responseHandler = handler
}

// UpdateTD updates the TD document of a consumed thing
// This returns the new consumed thing
// If the consumed thing doesn't exist then ignore this and return nil as
// it looks like it isn't used. ReadTD will read it if requested.
//
//	tdjson is the TD document in JSON format
func (cts *ConsumedThingsDirectory) UpdateTD(tdJSON string) *ConsumedThing {
	// convert the TD
	var tdi td.TD
	err := jsoniter.UnmarshalFromString(tdJSON, &tdi)
	if err != nil {
		return nil
	}
	cts.mux.Lock()
	defer cts.mux.Unlock()
	// Replace the TD in the consumed thing
	ct := cts.consumedThings[tdi.ID]
	if ct == nil {
		return nil
	}
	ct.tdi = &tdi
	return ct
}

// NewConsumedThingsSession creates a new instance of the session managing
// consumed Things through a client connection.
//
// This will subscribe to events from the Hub using the provided hub client.
func NewConsumedThingsSession(consumer *messaging.Consumer) *ConsumedThingsDirectory {
	ctm := ConsumedThingsDirectory{
		consumer:       consumer,
		consumedThings: make(map[string]*ConsumedThing),
		directory:      make(map[string]*td.TD),
	}
	//hc.Subscribe("", "") TODO: where to subscribe?
	consumer.SetResponseHandler(ctm.handleAsyncResponse)
	return &ctm
}
