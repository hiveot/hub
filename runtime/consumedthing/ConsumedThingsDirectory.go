package consumedthing

import (
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/consumer"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"sync"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
const ReadDirLimit = 1000

// ConsumedThingsDirectory manages a directory of consumed Things.
// Each consumed Thing holds a cache of a TD with latest property and event values.
//
// The cache must be updated by an external consumer that receives update events.
type ConsumedThingsDirectory struct {
	// the consumer connection for sending requests
	co *consumer.Consumer
	// Things used by the client
	consumedThings map[string]*ConsumedThing
	// directory of TD documents
	directory map[string]*td.TD
	// the full directory has been read in this session
	fullDirectoryRead bool
	mux               sync.RWMutex
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
	if !found {
		// create a new instance using the known TD, or request a TD from the server
		cts.mux.RLock()
		tdi, found2 := cts.directory[thingID]
		cts.mux.RUnlock()
		if !found2 {
			// request the TD from the Hub
			tdi, err = cts.ReadTD(thingID)
			if err != nil {
				return nil, err
			}
		}
		ct = NewConsumedThing(tdi, cts.co)
		_ = ct.Refresh()
		cts.mux.Lock()
		cts.consumedThings[thingID] = ct
		cts.mux.Unlock()
	}
	return ct, nil
}

// GetForm returns the form for an operation on a Thing
//func (cts *ConsumedThingsDirectory) GetForm(op, thingID, name string) (f *td.Form) {
//	tdi := cts.GetTD(thingID)
//	if tdi != nil {
//		f2 := tdi.GetForm(op, name, "")
//		f = &f2
//	}
//	return f
//}

// IsActive returns whether the session has a connection to the Hub or is in the process of connecting.
//func (cs *ConsumedThingsDirectory) IsActive() bool {
//	status := cs.co.GetStatus()
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

// OnResponse updates the consumed things from subscriptions
// To be invoked by the owner of the consumer when an async notification has
// been received.
func (cts *ConsumedThingsDirectory) OnResponse(msg *messaging.ResponseMessage) error {

	slog.Debug("ConsumedThingsDirectory.OnResponse",
		slog.String("senderID", msg.SenderID),
		slog.String("operation", msg.Operation),
		slog.String("thingID", msg.ThingID),
		slog.String("name", msg.Name),
		slog.String("clientID (me)", cts.co.GetClientID()),
	)
	ct, _ := cts.consumedThings[msg.ThingID]
	if ct != nil {
		ct.OnResponse(msg)
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
	thingsList, err := digitwin.ThingDirectoryReadAllTDs(cts.co, ReadDirLimit, 0)
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

// GetTD returns the cached TD of a thing
func (cts *ConsumedThingsDirectory) ReadTD(thingID string) (*td.TD, error) {
	// request the TD from the Hub
	tdi := &td.TD{}
	tdJson, err := digitwin.ThingDirectoryReadTD(cts.co, thingID)
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

//// SetResponseHandler registers a handler to receive async responses including events and property updates
//func (cts *ConsumedThingsDirectory) SetResponseHandler(handler func(message *transports.ResponseMessage)) {
//	cts.mux.Lock()
//	defer cts.mux.Unlock()
//	cts.responseHandler = handler
//}

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

// NewConsumedThingsDirectory returns a new instance of a directory for managing
// consumed Things for a client.
//
// This uses the consumer to send requests.
// The owner must call OnResponse() if an update is received.
//
// A consumed thing contains a Thing TD, with its property and event values. The
// values are updated when consumer events are received.
func NewConsumedThingsDirectory(co *consumer.Consumer) *ConsumedThingsDirectory {
	ctm := ConsumedThingsDirectory{
		co:             co,
		consumedThings: make(map[string]*ConsumedThing),
		directory:      make(map[string]*td.TD),
	}
	return &ctm
}
