package consumedthing

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"sync"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
const ReadDirLimit = 1000

// ConsumedThingsSession manages the consumed things of the Hub for a client session
//
// This maintains a single instance of each ConsumedThing and updates it when
// an event and action progress updates are received.
type ConsumedThingsSession struct {
	hc             hubclient.IHubClient
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
func (cts *ConsumedThingsSession) Consume(thingID string) (*ConsumedThing, error) {
	cts.mux.Lock()
	defer cts.mux.Unlock()
	ct, found := cts.consumedThings[thingID]
	if !found {
		// if the TD is known then use it
		td, found := cts.directory[thingID]
		if !found {
			// request the TD from the Hub
			td = &tdd.TD{}
			tdJson, err := digitwin.DirectoryReadTD(cts.hc, thingID)
			if err == nil {
				err = json.Unmarshal([]byte(tdJson), &td)
			}
			if err != nil {
				return nil, err
			}
			cts.directory[thingID] = td
		}
		ct = NewConsumedThing(td, cts.hc)
		ct.ReadAllEvents()
		ct.ReadAllProperties()
		cts.consumedThings[thingID] = ct
	}
	return ct, nil
}

// handleMessage updates the consumed things
func (cts *ConsumedThingsSession) handleMessage(
	msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {

	cts.mux.Lock()
	defer cts.mux.Unlock()
	if msg.MessageType == vocab.MessageTypeEvent {
		if msg.Key == vocab.EventTypeTD {
			// reload the TD
			td := &tdd.TD{}
			err := json.Unmarshal([]byte(msg.DataAsText()), &td)
			if err != nil {
				stat.Failed(msg, err)
				return stat
			}
			cts.directory[td.ID] = td
			// update consumed thing, if existing
			ct, found := cts.consumedThings[td.ID]
			if found {
				ct.td = td
				//ct.OnEvent(msg)
			}
			return
		}
		if msg.Key == vocab.EventTypeProperties {
			// update consumed thing, if existing
			ct, found := cts.consumedThings[msg.ThingID]
			if found {
				props := make(map[string]string)
				err := utils.DecodeAsObject(msg.Data, &props)
				if err == nil {
					// update all property values
					for k, v := range props {
						propMsg := hubclient.NewThingMessage(
							vocab.MessageTypeProperty, msg.ThingID, k, v, msg.SenderID)
						ct.OnPropertyUpdate(propMsg)
					}
				}
			}
			return
		}
		if msg.Key == vocab.EventTypeDeliveryUpdate {
			// delivery status updates refer to actions
			ct, found := cts.consumedThings[msg.ThingID]
			if found {
				ct.OnDeliveryUpdate(msg)
			}
			return
		}
		// this is a regular event
		ct, found := cts.consumedThings[msg.ThingID]
		if found {
			ct.eventValues[msg.Key] = msg
			ct.OnEvent(msg)
		}
		return
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
	thingsList, err := digitwin.DirectoryReadTDs(cts.hc, ReadDirLimit, 0)
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
func NewConsumedThingsSession(hc hubclient.IHubClient) *ConsumedThingsSession {
	ctm := ConsumedThingsSession{
		hc:             hc,
		consumedThings: make(map[string]*ConsumedThing),
		directory:      make(map[string]*tdd.TD),
	}
	hc.SetMessageHandler(ctm.handleMessage)
	return &ctm
}
