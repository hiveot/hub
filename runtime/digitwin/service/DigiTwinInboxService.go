package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"sync"
	"time"
)

// DigiTwinInboxService is the digital twin inbox for routing action requests to agents
// and sending the reply back to consumers.
//
// The typical action ingress flow is:
//
//	consumer -> protocol binding -> [digital twin inbox]
//
// Once received actions are forwarded to the agent of the destination Thing:
//
//	[digital twin inbox] -> protocol binding -> agent -> thing
//
// Agents respond with status update messages:
//
//	agent -> (delivery status) -> protocol binding -> [inbox]
//	[inbox] -> protocol binding -> consumer
type DigiTwinInboxService struct {

	// in-memory cache of active actions by messageID
	activeCache map[string]*digitwin.InboxRecord
	// in-memory latest actions by thingID.key
	latestByKey map[string]*digitwin.InboxRecord

	// protocol manager to send updates to clients
	pm api.ITransportBinding
	// mux to access the cache
	mux sync.RWMutex
}

// AddAction adds a new action request to the inbox.
func (svc *DigiTwinInboxService) AddAction(msg *hubclient.ThingMessage) (rec digitwin.InboxRecord, err error) {
	if msg.MessageID == "" || msg.ThingID == "" || msg.Key == "" || msg.SenderID == "" {
		err = fmt.Errorf(
			"action is missing required a parameter; senderID '%s', thingID '%s', key '%s', messageID '%s'",
			msg.SenderID, msg.ThingID, msg.Key, msg.MessageID)
		slog.Warn(err.Error())
		return
	}

	existing, err := svc.GetRecord(msg.MessageID)
	if err == nil {
		return existing, fmt.Errorf("MessageID already exists")
	}

	// this record holds the request information and the delivery progress
	// these are stored in the delivery inbox bucket.
	receivedTS := time.Now()
	timestamp := receivedTS.UnixMilli()
	record := &digitwin.InboxRecord{
		Input:       msg.Data,
		Key:         msg.Key,
		MessageID:   msg.MessageID,
		MessageType: msg.MessageType,
		Progress:    hubclient.DeliveredToInbox,
		Received:    receivedTS.Format(utils.RFC3339Milli),
		//Reply:       nil,   // don't store the reply. Not useful
		SenderID:  msg.SenderID,
		ThingID:   msg.ThingID,
		Timestamp: int(timestamp),
		Updated:   time.Now().Format(utils.RFC3339Milli),
	}

	svc.mux.Lock()
	svc.activeCache[record.MessageID] = record
	latestKey := msg.ThingID + "." + msg.Key
	svc.latestByKey[latestKey] = record
	svc.mux.Unlock()
	return *record, nil
}

// AddDeliveryStatus updates the delivery status of a message
func (svc *DigiTwinInboxService) AddDeliveryStatus(status hubclient.DeliveryStatus) error {
	record, err := svc.GetRecord(status.MessageID)
	if err == nil {
		record.Progress = status.Progress
		record.Error = status.Error
		record.Updated = time.Now().Format(utils.RFC3339Milli)
		svc.UpdateRecord(record)
	} else {
		err = fmt.Errorf("received DeliveryStatus update, but messageID '%s' is unknown", status.MessageID)
	}
	return err
}

// UpdateRecord stores an updated inbox record in the cache
// If the delivery status is completed or failed then it is removed from the in-memory cache
func (svc *DigiTwinInboxService) UpdateRecord(record digitwin.InboxRecord) {
	svc.mux.Lock()
	defer svc.mux.Unlock()
	if record.Progress == hubclient.DeliveryCompleted || record.Progress == hubclient.DeliveryFailed {
		// the action has completed. Remove it from the active cache
		// keep the latest cache for use by readLatest
		delete(svc.activeCache, record.MessageID)
	} else {
		svc.activeCache[record.MessageID] = &record
	}
	latestKey := record.ThingID + "." + record.Key
	svc.latestByKey[latestKey] = &record
}

// GetRecord returns the delivery status of a request by its messageID
func (svc *DigiTwinInboxService) GetRecord(messageID string) (r digitwin.InboxRecord, err error) {
	svc.mux.RLock()
	record, found := svc.activeCache[messageID]
	svc.mux.RUnlock()
	if !found {
		return r, fmt.Errorf("GetRecord, messageID '%s' not found", messageID)
	}
	return *record, nil
}

// HandleActionFlow receives a new action or property request from a consumer.
//
// This stores the action and forwards the request to the Thing's agent.
// This returns a possible reply and a delivery status. The reply is nil if the delivery
// is still in progress. If an error occurs then the delivery status contains the error.
//
// Action requests for the digital twin services directory, inbox and outbox
// are handled directly.
//
// Note that incoming action requests use the digital twin ThingID, not the physical
// device ID.
func (svc *DigiTwinInboxService) HandleActionFlow(msg *hubclient.ThingMessage) (status hubclient.DeliveryStatus) {
	slog.Info("inbox:HandleActionFlow",
		slog.String("ThingID", msg.ThingID),
		slog.String("Key", msg.Key),
		slog.String("SenderID", msg.SenderID),
		slog.String("MessageID", msg.MessageID),
	)
	// Store the service request in the svc and forward it to its agent
	// store the request. This already uses the digitwin thingID
	actionRecord, err := svc.AddAction(msg)
	if err != nil {
		slog.Error("HandleActionFlow failed",
			"err", err,
			"thingID", msg.ThingID)
	}

	// split the virtual thingID into the agent and serviceID
	// the agent is required to find the destination and the agent uses the native thingID (serviceID)
	DThingID := msg.ThingID
	agentID, serviceID := tdd.SplitDigiTwinThingID(DThingID)
	if agentID == "" {
		err = fmt.Errorf("agent '%s' for thing '%s' not found", agentID, msg.ThingID)
		slog.Warn(err.Error())
		actionRecord.Progress = hubclient.DeliveryFailed
		actionRecord.Error = err.Error()
		actionRecord.Updated = time.Now().Format(utils.RFC3339Milli)
		svc.UpdateRecord(actionRecord)
		return hubclient.DeliveryStatus{
			MessageID: msg.MessageID,
			Progress:  actionRecord.Progress,
			Error:     actionRecord.Error,
		}
	}

	// the message is forwarded to the agent
	msg.ThingID = serviceID
	actionRecord.Progress = hubclient.DeliveredToInbox

	stat, found := svc.pm.SendToClient(agentID, msg)
	if found {
		actionRecord.Progress = stat.Progress
		actionRecord.Delivered = time.Now().Format(utils.RFC3339Milli)
		actionRecord.Updated = actionRecord.Delivered
		svc.UpdateRecord(actionRecord)
	}

	// Progress updates from agents are sent as events and always asynchronously.
	return stat
}

// HandleDeliveryUpdate receives a delivery update event from agents.
// The message payload contains a DeliveryStatus object
//
// This updates the status of the inbox record and notifies the sender.
//
// If the message is no longer in the active cache then it is ignored.
func (svc *DigiTwinInboxService) HandleDeliveryUpdate(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
	var inboxRecord digitwin.InboxRecord

	err := utils.DecodeAsObject(msg.Data, &stat)
	if err == nil {
		inboxRecord, err = svc.GetRecord(stat.MessageID)
	}
	if err != nil {
		slog.Warn("HandleDeliveryUpdate: Message not in active cache. It is ignored")
		// the delivery update is completed by ignoring the message
		stat.Completed(msg, nil, err)
		return stat
	}

	// error checking that the update does belong to the right thing action
	slog.Info("inbox:HandleDeliveryUpdate ",
		slog.String("ThingID", inboxRecord.ThingID),
		slog.String("Key", inboxRecord.Key),
		slog.String("Progress", stat.Progress),
		slog.String("error", stat.Error),
		slog.String("MessageID", stat.MessageID),
	)

	// the sender (agents) must be the thing agent
	thingAgentID, thingID := tdd.SplitDigiTwinThingID(inboxRecord.ThingID)
	_ = thingID
	if thingAgentID != msg.SenderID {
		err = fmt.Errorf("inbox:HandleDeliveryUpdate: status update '%s' of thing '%s' does not come from agent '%s' but from '%s'. Update ignored.",
			stat.MessageID, msg.ThingID, thingAgentID, msg.SenderID)
	}

	if err == nil {
		// update the action delivery status
		err = svc.AddDeliveryStatus(stat)
		// notify the action sender of the delivery update
		msg2 := *msg
		svc.pm.SendToClient(inboxRecord.SenderID, &msg2)
	}
	if err != nil {
		slog.Warn("inbox:HandleDeliveryUpdate",
			slog.String("senderID", msg.SenderID),
			slog.String("thingID", msg.ThingID),
			slog.String("err", err.Error()),
			slog.String("MessageID", stat.MessageID),
		)
		err = nil
	}
	// the delivery update itself has completed
	stat.Completed(msg, nil, err)
	return stat
}

// ReadLatest returns the most recent inbox record of a thing's action
func (svc *DigiTwinInboxService) ReadLatest(
	senderID string, args digitwin.InboxReadLatestArgs) (record digitwin.InboxRecord, err error) {

	latestKey := args.ThingID + "." + args.Key
	svc.mux.RLock()
	r, found := svc.latestByKey[latestKey]
	if found {
		record = *r
	} else {
		err = fmt.Errorf("ReadLatest: Inbox does not have the latest action record for thing/key '%s/%s'", args.ThingID, args.Key)
	}
	svc.mux.RUnlock()
	return record, err
}

func (svc *DigiTwinInboxService) Start() error {
	return nil
}

// Stop clears the cache
func (svc *DigiTwinInboxService) Stop() {
	svc.mux.Lock()
	svc.activeCache = make(map[string]*digitwin.InboxRecord)
	svc.latestByKey = make(map[string]*digitwin.InboxRecord)
	svc.mux.Unlock()
}

// NewDigiTwinInbox returns a new instance of the inbox service.
// the store to persist the cache between restarts - not currently used
// tb is the protocolbinding api for sending clients delivery status messages
func NewDigiTwinInbox(bucketStore buckets.IBucketStore, pm api.ITransportBinding) *DigiTwinInboxService {
	dtInbox := &DigiTwinInboxService{
		activeCache: make(map[string]*digitwin.InboxRecord),
		latestByKey: make(map[string]*digitwin.InboxRecord),
		pm:          pm,
	}
	return dtInbox
}
