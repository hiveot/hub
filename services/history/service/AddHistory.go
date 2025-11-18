package service

import (
	"fmt"
	"log/slog"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hivekit/go/wot"
	"github.com/hiveot/hub/lib/buckets"
	jsoniter "github.com/json-iterator/go"
)

const DefaultMaxMessageSize = 30

// AddHistory adds events and actions of any Thing
type AddHistory struct {
	// store with a bucket for each Thing
	store buckets.IBucketStore
	// onAddedValue is a callback to invoke after a value is added. Intended for tracking most recent values.
	//onAddedValue func(ev *transports.IConsumer)
	//
	retentionMgr *ManageHistory
	// Maximum message size in bytes.
	MaxMessageSize int
}

// encode a ResponseMessage into a single storage key value pair for easy storage and filtering.
// Encoding generates a key as: timestampMsec/name/a|e|p/sender,
// where a|e|p indicates message type "action", "event" or "property"
func (svc *AddHistory) encodeValue(senderID string, tv *messaging.ThingValue) (storageKey string, data []byte) {
	var err error
	createdTime := time.Now().UTC()
	if tv.Timestamp != "" {
		createdTime, err = dateparse.ParseAny(tv.Timestamp)
		if err != nil {
			slog.Warn("Invalid Timestamp time. Using current time instead", "created", tv.Timestamp)
			createdTime = time.Now().UTC()
		}
	}

	// the index uses milliseconds for timestamp
	timestamp := createdTime.UnixMilli()
	storageKey = strconv.FormatInt(timestamp, 10) + "/" + tv.Name
	if tv.AffordanceType == messaging.AffordanceTypeAction {
		// TODO: actions subscriptions are currently not supported. This would be useful though.
		storageKey = storageKey + "/a"
	} else if tv.AffordanceType == messaging.AffordanceTypeProperty {
		storageKey = storageKey + "/p"
	} else { // treat everything else as events
		storageKey = storageKey + "/e"
	}
	storageKey = storageKey + "/" + senderID
	//if msg.Data != nil {
	data, _ = jsoniter.Marshal(tv.Data)
	//}
	return storageKey, data
}

// AddValue adds a Thing value from a sender to the action history
func (svc *AddHistory) AddValue(senderID string, tv messaging.ThingValue) error {
	//slog.Info("AddValue",
	//	slog.String("senderID", senderID),
	//	slog.String("ID", tv.ID),
	//	slog.String("thingID", tv.ThingID),
	//	slog.String("name", tv.Name),
	//	slog.String("affordance", tv.AffordanceType),
	//)
	retain, err := svc.validateValue(senderID, &tv)
	if err != nil {
		slog.Info("AddValue value error", "err", err.Error())
		return err
	}
	if !retain {
		slog.Debug("AddValue value not retained",
			slog.String("name", tv.Name))
		return nil
	}
	storageKey, val := svc.encodeValue(senderID, &tv)
	bucket := svc.store.GetBucket(tv.ThingID)
	err = bucket.Set(storageKey, val)
	_ = bucket.Close()
	//if svc.onAddedValue != nil {
	//	svc.onAddedValue(actionValue)
	//}
	return err
}

// AddMessage adds the value of an event, action or property notification to the history store
func (svc *AddHistory) AddMessage(msg *messaging.NotificationMessage) error {
	// FIXME: store the action request or response?
	// How to obtain the action request?
	// How to subscribe to action responses?
	// Option1: digitwin allows subscribing to action responses (HTOpSubscribeAction?) - not compatible, but thats okay?
	// Option2: digitwin publishes an actionstatus event (OpSubscribeEvent) - thingID won't match
	// Option3: digitwin publishes the last action response as notification - not a TD event
	tv := messaging.ThingValue{
		//ID:      msg.CorrelationID,
		Name:      msg.Name,
		Data:      msg.Value,
		ThingID:   msg.ThingID,
		Timestamp: msg.Timestamp,
	}
	switch msg.Operation {
	case wot.OpInvokeAction:
		tv.AffordanceType = messaging.AffordanceTypeAction
		return svc.AddValue(msg.SenderID, tv) // response of an action
	case wot.OpObserveProperty:
		tv.AffordanceType = messaging.AffordanceTypeProperty
		return svc.AddValue(msg.SenderID, tv)
	case wot.OpObserveAllProperties:
		// output is a key:value map
		tv.AffordanceType = messaging.AffordanceTypeProperty
		propMap := make(map[string]any)
		err := utils.DecodeAsObject(msg.Value, &propMap)
		if err != nil {
			return err
		}
		for k, v := range propMap {
			tv.Name = k
			tv.Data = v
			_ = svc.AddValue(msg.SenderID, tv)
		}
		return err
	case wot.OpSubscribeEvent, wot.OpSubscribeAllEvents:
		tv.AffordanceType = messaging.AffordanceTypeEvent
		return svc.AddValue(msg.SenderID, tv)
	default:
		// anything else is added as an event
		tv.AffordanceType = messaging.AffordanceTypeEvent
		return svc.AddValue(msg.SenderID, tv)
	}
}

// validateValue checks the event has the right things address, adds a timestamp if missing and returns if it is retained
// an error will be returned if the agentID, thingID or name are empty.
// retained returns true if the value is valid and passes the retention rules
func (svc *AddHistory) validateValue(senderID string, tv *messaging.ThingValue) (retained bool, err error) {
	if tv.ThingID == "" {
		return false, fmt.Errorf("missing thingID in value with value name '%s'", tv.Name)
	}
	if tv.Name == "" {
		return false, fmt.Errorf("missing name for event or action for things '%s'", tv.ThingID)
	}
	if senderID == "" && tv.AffordanceType == messaging.AffordanceTypeAction {
		return false, fmt.Errorf("missing sender for action on thing '%s'", tv.ThingID)
	}
	if tv.Timestamp == "" {
		tv.Timestamp = utils.FormatNowUTCMilli()
	}
	if svc.retentionMgr != nil {
		retain, rule := svc.retentionMgr._IsRetained(tv)
		if rule == nil {
			slog.Debug("no retention rule found for event",
				slog.String("name", tv.Name), slog.Bool("retain", retain))
		}
		return retain, nil
	}

	return true, nil
}

// NewAddHistory provides the capability to add values to Thing history buckets
//
//	store with a bucket for each Thing
//	retentionMgr is optional and used to apply constraints to the events to add
func NewAddHistory(
	store buckets.IBucketStore, retentionMgr *ManageHistory) *AddHistory {
	svc := &AddHistory{
		store:          store,
		retentionMgr:   retentionMgr,
		MaxMessageSize: DefaultMaxMessageSize,
	}

	return svc
}
