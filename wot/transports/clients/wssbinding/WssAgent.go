// package wssclient with outgoing requests
package wssbinding

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/transports"
	"github.com/teris-io/shortid"
	"log/slog"
	"time"
)

// PubActionStatus agent publishes an action progress message to the digital twin.
// The digital twin will update the request status and notify the sender.
// This returns an error if the connection with the server is broken
func (cl *WssBindingClient) PubActionStatus(stat transports.RequestStatus) {
	slog.Debug("PubActionStatus",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", stat.ThingID),
		slog.String("name", stat.Name),
		slog.String("progress", stat.Status),
		slog.String("requestID", stat.CorrelationID))

	msg := ActionStatusMessage{
		MessageType:   MsgTypeActionStatus,
		ThingID:       stat.ThingID,
		Name:          stat.Name,
		CorrelationID: stat.CorrelationID,
		MessageID:     shortid.MustGenerate(),
		Status:        stat.Status,
		Error:         stat.Error,
		Output:        stat.Output,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	_ = cl._send(msg)
}

// PubEvent agent publishes an event message and returns
// This returns an error if the connection with the server is broken
func (cl *WssBindingClient) PubEvent(
	thingID string, name string, data any, correlationID string) error {

	slog.Debug("PubEvent",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("data", data),
		//slog.String("requestID", requestID),
	)
	msg := EventMessage{
		ThingID:       thingID,
		MessageType:   MsgTypePublishEvent,
		Name:          name,
		Data:          data,
		CorrelationID: correlationID,
		Timestamp:     time.Now().Format(utils.RFC3339Milli),
	}
	err := cl._send(msg)
	return err
}

// PubMultipleProperties agent updates a batch of property values.
// Intended for use by agents
func (cl *WssBindingClient) PubMultipleProperties(thingID string, propMap map[string]any) error {
	slog.Info("PubMultipleProperties",
		slog.String("thingID", thingID),
		slog.Int("nr props", len(propMap)),
	)

	msg := PropertyMessage{
		ThingID:     thingID,
		MessageType: MsgTypePropertyReadings,
		Data:        propMap,
		Timestamp:   time.Now().Format(utils.RFC3339Milli),
		MessageID:   shortid.MustGenerate(),
	}
	err := cl._send(msg)
	return err
}

// PubProperty agent sends an update of a property value.
// Intended for use by agents
func (cl *WssBindingClient) PubProperty(thingID string, name string, value any) error {
	slog.Info("PubProperty",
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("value", value))

	msg := PropertyMessage{
		ThingID:     thingID,
		MessageType: MsgTypePropertyReading,
		Name:        name,
		Data:        value,
		Timestamp:   time.Now().Format(utils.RFC3339Milli),
		MessageID:   shortid.MustGenerate(),
	}
	err := cl._send(msg)
	return err
}

// PubTD publishes a TD update.
// This is short for a digitwin directory updateTD action
// TODO: this approach to updating the TD is tentative. A couple of notes:
// - WoT points to using discovery for obtaining the directory.
// - TDs are updated rarely. Is there a need for a push model?
// - When an agent connects to hiveot hub. How to get the TD?
//   - ignore the connection and use DNS-SD or something like it.
//     with a zwavejs agent you can easily have over 200 TDs
//   - all that is needed is an API to either push or pull TDs
//     does discovery supports using an existing connection to read all TDs?
//     this is also needed by consumers after all.
func (cl *WssBindingClient) PubTD(thingID string, tdJSON string) error {
	slog.Info("PubTD", slog.String("thingID", thingID))

	msg := TDMessage{
		ThingID:     thingID,
		MessageType: MsgTypeUpdateTD,
		Name:        "",
		Data:        tdJSON,
		Timestamp:   time.Now().Format(utils.RFC3339Milli),
		MessageID:   shortid.MustGenerate(),
	}
	err := cl._send(msg)

	return err
}
