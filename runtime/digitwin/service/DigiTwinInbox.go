package service

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/inbox"
	"github.com/hiveot/hub/api/go/outbox"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"log/slog"
	"time"
)

const ActionRecordsBucketName = "actionHistory"
const LatestActionsBucketName = "latestActions"

// InboxRecord contains the action request record with delivery status and possibly a reply.
type InboxRecord struct {
	// The message to deliver. This contains the digital twin's thingID.
	Request things.ThingMessage `json:"request"`
	// The current delivery status
	DeliveryStatus api.DeliveryStatus `json:"deliveryStatus"`
	// Time the request was delivered to the agent
	DeliveredMSec int64 `json:"delivered"`
	// Time the request was received
	ReceivedMSec int64 `json:"received"`
	// Time of last status update as milli-seconds since epoch
	UpdatedMSec int64 `json:"updated"`
}

// DigiTwinInbox is the digital twin inbox for storing actions sent to the digital twin by consumers.
//
// The typical action ingress flow is:
//
//	consumer -> protocol binding -> router -> [digital twin inbox]
//
// Once received actions are forwarded to the agent of the destination Thing:
//
//	[digital twin inbox] -> router -> protocol binding -> agent -> thing
//
// Agents respond with status update messages
type DigiTwinInbox struct {
	// The inbox storage buckets with action records by message ID.
	actionRecords buckets.IBucket
	// latest actions store for reading the last received actions
	latest *DigiTwinLatestStore
	// protocol manager to send updates to clients
	pm api.ITransportBinding
}

// AddAction adds a new action to the inbox
func (svc *DigiTwinInbox) AddAction(msg *things.ThingMessage) (InboxRecord, error) {
	record := InboxRecord{
		// store a copy of the message
		Request: *msg,
		DeliveryStatus: api.DeliveryStatus{
			MessageID: msg.MessageID,
			Status:    api.DeliveryPending,
			Reply:     nil,
		},
		ReceivedMSec:  time.Now().UnixMilli(),
		DeliveredMSec: 0,
		UpdatedMSec:   0,
	}
	recordJSON, _ := json.Marshal(record)
	err := svc.actionRecords.Set(record.DeliveryStatus.MessageID, recordJSON)
	return record, err
}

// GetRecord returns the delivery status of a request
func (svc *DigiTwinInbox) GetRecord(messageID string) (r InboxRecord, err error) {
	value, err := svc.actionRecords.Get(messageID)
	if err == nil {
		err = json.Unmarshal(value, &r)
	}
	return r, err
}

// HandleActionFlow receives a new action request from a consumer.
// This stores the action and forwards the request to the Thing's agent.
// This returns a possible reply and a delivery status. The reply is nil if the delivery
// is still in progress. If an error occurs then the delivery status contains the error.
//
// Action requests for the digital twin services directory, inbox and outbox
// are handled directly.
//
// Note that incoming action requests use the ThingID of virtual device, not the physical
// device.
func (svc *DigiTwinInbox) HandleActionFlow(msg *things.ThingMessage) (status api.DeliveryStatus) {
	// all latest values are stored
	svc.latest.StoreMessage(msg)

	// Store the service request in the svc and forward it to its agent
	// store the request. This already uses the digitwin thingID
	actionRecord, err := svc.AddAction(msg)
	if err != nil {
		slog.Error("HandleActionFlow failed",
			"err", err,
			"thingID", msg.ThingID)
	}

	// split the virtual thingID into the agent and serviceID
	// the agent is needed to find the destination and the agent uses the native thingID (serviceID)
	DThingID := msg.ThingID
	agentID, serviceID := things.SplitDigiTwinThingID(DThingID)
	if agentID == "" {
		actionRecord.DeliveryStatus.Status = api.DeliveryFailed
		actionRecord.DeliveryStatus.Error = fmt.Sprintf("Agent for thing '%s' not found", msg.ThingID)
		return actionRecord.DeliveryStatus
	}
	// the message itself is forwarded to the agent using the device's service
	msg.ThingID = serviceID
	actionRecord.DeliveryStatus.Status = api.DeliveryPending

	stat, _ := svc.pm.SendToClient(agentID, msg)
	actionRecord.DeliveryStatus = stat

	// Status updates from agents are sent as events and always asynchronously.
	//
	// TODO: TBD use channels with a timeout to wait for potential immediate reply or have this
	// handled by the transport?
	return actionRecord.DeliveryStatus
}

// HandleDeliveryUpdate receives a delivery update event from agents.
// The message payload contains a DeliveryStatus object
//
// This updates the status of the inbox record and notifies the sender.
func (svc *DigiTwinInbox) HandleDeliveryUpdate(msg *things.ThingMessage) api.DeliveryStatus {
	slog.Info("HandleDeliveryUpdate",
		slog.String("thingID", msg.ThingID),
		slog.String("MessageID", msg.MessageID),
		slog.String("SenderID", msg.SenderID))

	var inboxRecord InboxRecord
	newStatus := api.DeliveryStatus{}
	err := json.Unmarshal(msg.Data, &newStatus)
	if err == nil {
		inboxRecord, err = svc.GetRecord(newStatus.MessageID)
	}
	// error checking that the update does belong to the right thing action
	if err == nil {
		// the sender (agents) must match
		thingAgentID, thingID := things.SplitDigiTwinThingID(inboxRecord.Request.ThingID)
		if thingAgentID != msg.SenderID {
			err = fmt.Errorf("HandleDeliveryUpdate: status update '%s' of thing '%s' does not come from agent '%s' but from '%s'. Update ignored.",
				newStatus.MessageID, msg.ThingID, thingAgentID, msg.SenderID)
		} else if thingID != msg.ThingID {
			err = fmt.Errorf("HandleDeliveryUpdate: status update '%s' of thing '%s' does not match the thingID '%s' in the inbox. Update ignored.",
				newStatus.MessageID, msg.ThingID, thingAgentID)
		}
	}
	if err == nil {
		// update the action delivery status
		err = svc.SetStatus(newStatus)
		// notify the action sender of the delivery update
		msg2 := *msg
		svc.pm.SendToClient(inboxRecord.Request.SenderID, &msg2)
	}
	if err != nil {
		slog.Warn(err.Error())
	}
	// the delivery update is delivered
	return api.DeliveryStatus{MessageID: msg.MessageID, Status: api.DeliveryCompleted}
}

// NotifyStatus sends a delivery status message to the consumer
func (svc *DigiTwinInbox) NotifyStatus(messageID string) {

}

// ReadLatest returns the latest value of each action of a thing
func (svc *DigiTwinInbox) ReadLatest(args inbox.ReadLatestArgs) (inbox.ReadLatestResp, error) {
	recs, err := svc.latest.ReadLatest(vocab.MessageTypeAction, args.ThingID, nil, "")
	resp := inbox.ReadLatestResp{ThingValues: recs}
	return resp, err
}

// RemoveValue Remove Thing action value
// Intended to remove outliers
func (svc *DigiTwinInbox) RemoveValue(args outbox.RemoveValueArgs) error {
	return fmt.Errorf("not yet implemented")
}

// SetStatus updates the delivery status of a request
func (svc *DigiTwinInbox) SetStatus(status api.DeliveryStatus) error {
	record, err := svc.GetRecord(status.MessageID)
	if err == nil {
		record.DeliveryStatus = status
		recordJSON, _ := json.Marshal(record)
		err = svc.actionRecords.Set(status.MessageID, recordJSON)
	}
	// FIXME: move completed/failed records to the inactive records
	// or, use a single store and be smart about querying
	return err
}

func (svc *DigiTwinInbox) Start() error {
	return nil
}
func (svc *DigiTwinInbox) Stop() {
	_ = svc.actionRecords.Close()
}

// NewDigiTwinInbox returns a new instance of the inbox service using the given storage bucket
// pm is the protocolbinding api for sending clients delivery status messages
func NewDigiTwinInbox(bucketStore buckets.IBucketStore, pm api.ITransportBinding) *DigiTwinInbox {
	actionsBucket := bucketStore.GetBucket(ActionRecordsBucketName)
	latestBucket := bucketStore.GetBucket(LatestActionsBucketName)
	latestStore := NewDigiTwinLatestStore(latestBucket)
	dtInbox := &DigiTwinInbox{
		actionRecords: actionsBucket,
		latest:        latestStore,
		pm:            pm,
	}
	return dtInbox
}
