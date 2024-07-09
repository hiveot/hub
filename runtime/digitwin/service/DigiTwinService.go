package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"path"
	"sync"
)

// The DigitwinService orchestrates the flow of events and actions with Thing agents and consumers
// It manages storage of events, actions and communicates with agents and consumers using the
// protocol manager. It uses a helper to manage the mapping of things to the agents that serve them.
type DigitwinService struct {
	// underlying store for the directory, inbox and outbox
	store buckets.IBucketStore

	// The directory stores digitwin TDD documents
	Directory *DigitwinDirectoryService
	// The inbox handles incoming action requests from consumers
	Inbox *DigiTwinInboxService
	// The outbox receives events from agents and can be queried by consumers
	Outbox *DigiTwinOutboxService

	mux sync.RWMutex
	// The protocol manager communicates with agents and consumers
	pm api.ITransportBinding
}

// HandleMessage is the main ingress point of the messages flow to the digital twin entities.
// * Actions are passed to the inbox for processing
// * Delivery update events are passed to the inbox for updating action status
// * TD events are passed to the directory for updating the directory
// * All remaining events are passed to the outbox for distribution to subscribers
//
// Note that this is separate from access to the API's of directory, inbox and outbox,
// which is handled by the digitwin agent.
//
// In this case the service wears two hats, one to process and direct the message flow (this handler)
// and second, to give clients access to the digitwin API, which is handled by the agent.
func (svc *DigitwinService) HandleMessage(msg *things.ThingMessage) (stat hubclient.DeliveryStatus) {
	// action request go to the inbox to be passed on to the destination
	if msg.MessageType == vocab.MessageTypeAction {
		return svc.Inbox.HandleActionFlow(msg)
	}
	if msg.MessageType == vocab.MessageTypeProperty {
		return svc.Inbox.HandleActionFlow(msg)
	}
	// action delivery update event, send by client
	if msg.Key == vocab.EventTypeDeliveryUpdate {
		return svc.Inbox.HandleDeliveryUpdate(msg)
	}
	// TD event updates the directory and are broadcast to subscribers
	if msg.Key == vocab.EventTypeTD {
		svc.Directory.HandleTDEvent(msg)
	}
	// regular events to be broadcast to subscribers
	return svc.Outbox.HandleEvent(msg)
}

// Start the digitwin service with inbox, outbox and Thing directory
func (svc *DigitwinService) Start() (err error) {
	slog.Info("Starting DigitwinService")
	err = svc.store.Open()
	svc.Inbox = NewDigiTwinInbox(svc.store, svc.pm)
	svc.Outbox = NewDigiTwinOutbox(svc.store, svc.pm)
	svc.Directory = NewDigitwinDirectory(svc.store)
	if err == nil {
		err = svc.Directory.Start()
	}
	if err == nil {
		err = svc.Outbox.Start()
	}
	if err == nil {
		err = svc.Inbox.Start()
	}

	return err
}

// Stop the service
func (svc *DigitwinService) Stop() {
	svc.Outbox.Stop()
	svc.Inbox.Stop()
	svc.Directory.Stop()
	slog.Info("Stopping DigitwinService")
	_ = svc.store.Close()
}

// NewDigitwinService creates a new instance of the Digitwin service
// The digitwin service is responsible for representing a Thing to consumers.
//
//	pm is the protocol manager used to communicate with agents and consumers
//	store is the bucket store for inbox and outbox storage. It will be opened on start and closed on stop
func NewDigitwinService(pm api.ITransportBinding, store buckets.IBucketStore) *DigitwinService {
	svc := &DigitwinService{
		store: store,
		pm:    pm,
		mux:   sync.RWMutex{},
	}
	return svc
}

// StartDigitwinService creates and start the digitwin administration service.
// This creates a bucket store for the directory, inbox, and outbox.
//
// storesDir is the directory where to create the digitwin storage
// pm is the protocol binding or manager used to send messages to clients
func StartDigitwinService(
	storesDir string, pm api.ITransportBinding) (svc *DigitwinService, err error) {

	var store buckets.IBucketStore

	storePath := path.Join(storesDir, "digitwin", "digitwin.store")
	store = kvbtree.NewKVStore(storePath)
	if err == nil {
		svc = NewDigitwinService(pm, store)
		err = svc.Start()
	}
	return svc, err
}
