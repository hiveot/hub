package consumedthing

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"sync"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
const ReadDirLimit = 1000

// ConsumedThingsSession manages the consumed things of the Hub for a client session
//
// This maintains a single instance of each ConsumedThing and updates it when
// an event and action progress updates are received.
type ConsumedThingsSession struct {
	hc hubclient.IConsumerClient
	// Things used by the client
	consumedThings map[string]*ConsumedThing
	// directory of TD documents
	directory map[string]*tdd.TD
	// the full directory has been read in this session
	fullDirectoryRead bool
	// additional handler of events for forwarding to other consumers
	eventHandler func(msg *hubclient.ThingMessage)
	mux          sync.RWMutex
}

// Consume creates a ConsumedThing instance for reading and operating a Thing.
//
// If an instance already exists it is returned, otherwise one is created using
// the ThingDescription. If the TD is unknown then request it from the directory
// This returns an error if a valid Thing cannot be found.
func (cts *ConsumedThingsSession) Consume(thingID string) (ct *ConsumedThing, err error) {
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
		ct = NewConsumedThing(td, cts.hc)
		ct.ReadAllEvents()
		ct.ReadAllProperties()
		cts.mux.Lock()
		cts.consumedThings[thingID] = ct
		cts.mux.Unlock()
	}
	return ct, nil
}

// handleMessage updates the consumed things
func (cts *ConsumedThingsSession) handleMessage(
	msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {

	// if an event is received from an unknown Thing then (re)load its TD
	cts.mux.RLock()
	_, found := cts.directory[msg.ThingID]
	cts.mux.RUnlock()
	if !found {
		_, err := cts.ReadTD(msg.ThingID)
		if err != nil {
			slog.Error("Received message with thingID that doesn't exist",
				"thingID", msg.ThingID, "name", msg.Name, "senderID", msg.SenderID)
		}
	}

	if msg.MessageType == vocab.MessageTypeTD {
		// reload the TD
		td := &tdd.TD{}
		err := json.Unmarshal([]byte(msg.DataAsText()), &td)
		if err != nil {
			slog.Error("invalid payload for TD event. Ignored",
				"thingID", msg.ThingID)
			stat.Failed(msg, err)
			return stat
		}
		cts.mux.Lock()
		defer cts.mux.Unlock()
		cts.directory[td.ID] = td
		// update consumed thing, if existing
		ct, found := cts.consumedThings[td.ID]
		if found {
			ct.td = td
			//ct.OnEvent(msg)
		}
	} else if msg.MessageType == vocab.MessageTypeProperty {
		// update consumed thing, if existing
		cts.mux.Lock()
		defer cts.mux.Unlock()
		ct, found := cts.consumedThings[msg.ThingID]
		if found {
			props := make(map[string]interface{})
			err := utils.DecodeAsObject(msg.Data, &props)
			if err != nil {
				slog.Error("invalid payload for properties event. Ignored",
					"thingID", msg.ThingID)
			} else {
				// update all property values
				for k, v := range props {
					propValue := &digitwin.ThingValue{
						Name: k, Data: v, MessageID: msg.MessageID,
						SenderID: msg.SenderID, Updated: msg.Created}
					ct.OnPropertyUpdate(propValue)
				}
			}
		}
	} else if msg.MessageType == vocab.MessageTypeDeliveryUpdate {
		// delivery status updates refer to actions
		cts.mux.RLock()
		ct, found := cts.consumedThings[msg.ThingID]
		cts.mux.RUnlock()
		if found {
			ct.OnDeliveryUpdate(msg)
		}
	} else {
		// this is a regular value event
		cts.mux.RLock()
		ct, found := cts.consumedThings[msg.ThingID]
		cts.mux.RUnlock()
		if found {
			tv := &digitwin.ThingValue{
				Name: msg.Name, MessageID: msg.MessageID, SenderID: msg.SenderID,
				Updated: msg.Created, Data: msg.Data}
			ct.OnEvent(tv)
		}

	}
	// pass it on to chained handler
	if cts.eventHandler != nil {
		cts.eventHandler(msg)
	}
	return stat
}

// IsActive returns whether the session has a connection to the Hub or is in the process of connecting.
//func (cs *ConsumedThingsSession) IsActive() bool {
//	status := cs.hc.GetStatus()
//	return status.ConnectionStatus == hubclient.Connected ||
//		status.ConnectionStatus == hubclient.Connecting
//}

// GetTD returns the TD in the session or nil if thingID isn't found
func (cts *ConsumedThingsSession) GetTD(thingID string) *tdd.TD {
	cts.mux.RLock()
	defer cts.mux.RUnlock()
	td := cts.directory[thingID]
	return td
}

// ReadDirectory loads and decodes Thing Description documents from the directory.
// These are used by Consume()
// This currently limits the nr of things to ReadDirLimit.
//
// Set force to force a reload of the directory instead of using any cached TD documents.
// TODO: reload when things were added or removed
func (cts *ConsumedThingsSession) ReadDirectory(force bool) (map[string]*tdd.TD, error) {
	cts.mux.RLock()
	fullDirectoryRead := cts.fullDirectoryRead
	currentDir := cts.directory
	cts.mux.RUnlock()

	// return the cached version
	if !force && fullDirectoryRead {
		return currentDir, nil
	}
	newDir := make(map[string]*tdd.TD)

	// TODO: support for reading in pages
	thingsList, err := digitwin.DirectoryReadAllDTDs(cts.hc, ReadDirLimit, 0)
	if err != nil {
		return newDir, err
	}
	for _, tdJson := range thingsList {
		td := tdd.TD{}
		err = json.Unmarshal([]byte(tdJson), &td)
		if err == nil {
			newDir[td.ID] = &td
		}
	}
	cts.mux.Lock()
	defer cts.mux.Unlock()
	cts.fullDirectoryRead = true
	cts.directory = newDir
	return newDir, nil
}

// ReadTD reads a TD from the directory service and updates the directory cache.
func (cts *ConsumedThingsSession) ReadTD(thingID string) (*tdd.TD, error) {
	// request the TD from the Hub
	td := &tdd.TD{}
	tdJson, err := digitwin.DirectoryReadDTD(cts.hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJson), &td)
	}
	if err != nil {
		return nil, err
	}
	cts.mux.Lock()
	defer cts.mux.Unlock()
	cts.directory[thingID] = td
	return td, err
}

// SetEventHandler registers a handler to receive events in addition
// to the consumed things themselves.
// Intended for consumers such that need to pass all messages on, for example to
// a UI frontend.
// Currently only a single handler is supported.
func (cts *ConsumedThingsSession) SetEventHandler(handler func(message *hubclient.ThingMessage)) {
	cts.mux.Lock()
	defer cts.mux.Unlock()
	cts.eventHandler = handler
}

// NewConsumedThingsSession creates a new instance of the session managing
// consumed Things through a client connection.
//
// This will subscribe to events from the Hub using the provided hub client.
// This will receive any prior event subscriber.
func NewConsumedThingsSession(hc hubclient.IConsumerClient) *ConsumedThingsSession {
	ctm := ConsumedThingsSession{
		hc:             hc,
		consumedThings: make(map[string]*ConsumedThing),
		directory:      make(map[string]*tdd.TD),
	}
	hc.SetMessageHandler(ctm.handleMessage)
	return &ctm
}
