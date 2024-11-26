// Package httpbinding with operations for agents.
// Since the TD is intended for consumers, it does not describe how agents
// can post messages when connected as a client to a hub.
package httpbinding

import (
	"errors"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/transports"
	"log/slog"
	"net/http"
)

// Hub paths for use by Thing agents.
// TODO: find a form-based way to obtain these
// The plan is for digitwin to publish an 'agent service' TD for Things to publish
// their information instead of expecting consumers to connect to them.
// Agent clients (eg, this one) will discover the hub address, retrieve its TDs
// and prime themselves for use using the found TDs.
//
// In the interim, the simpler solution is to simply hardcode the Forms to match
// those of the http server.
//
// TODO2: what operations to use?
//   - define hiveot custom operations
//   - define custom actions in the digitwin TD. pretty much the same idea.
const (
	PostAgentPublishEventPath             = "/agent/event/{thingID}/{name}"
	PostAgentPublishProgressPath          = "/agent/progress"
	PostAgentUpdatePropertyPath           = "/agent/property/{thingID}/{name}"
	PostAgentUpdateMultiplePropertiesPath = "/agent/properties/{thingID}"
	PostAgentUpdateTDDPath                = "/agent/tdd/{thingID}"
)

// PubActionStatus agent publishes an action status update message over http
// The digital twin will update the request status and notify the sender.
// This returns an error if the connection with the server is broken
func (cl *HttpBindingClient) PubActionStatus(stat transports.RequestStatus) {
	// return the status of the invoke-action operation
	stat.Operation = vocab.OpInvokeAction
	cl.SendOperationStatus(stat)
}

// PubEvent publishes an event message and returns
// This returns an error if the connection with the server is broken
func (cl *HttpBindingClient) PubEvent(
	thingID string, name string, data any, CorrelationID string) error {

	slog.Debug("PubEvent",
		slog.String("agentID", cl.clientID),
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("data", data),
		//slog.String("requestID", requestID),
	)
	stat := cl.Pub(http.MethodPost, PostAgentPublishEventPath,
		thingID, name, data, CorrelationID)
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubMultipleProperties agent publishes a batch of property values.
// Intended for use by agents
func (cl *HttpBindingClient) PubMultipleProperties(thingID string, propMap map[string]any) error {
	slog.Info("PubMultipleProperties",
		slog.String("thingID", thingID),
		slog.Int("nr props", len(propMap)),
	)
	stat := cl.Pub(http.MethodPost, PostAgentUpdateMultiplePropertiesPath,
		thingID, "", propMap, "")
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubProperty agent publishes a property value update.
// Intended for use by agents to property changes
func (cl *HttpBindingClient) PubProperty(thingID string, name string, value any) error {
	slog.Info("PubProperty",
		slog.String("thingID", thingID),
		slog.String("name", name),
		slog.Any("value", value))

	stat := cl.Pub("POST", PostAgentUpdatePropertyPath,
		thingID, name, value, "")
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}

// PubTD publishes a TD update.
// This is short for a digitwin directory updateTD action
func (cl *HttpBindingClient) PubTD(thingID string, tdJSON string) error {
	slog.Info("PubTD", slog.String("thingID", thingID))

	stat := cl.Pub("POST", PostAgentUpdateTDDPath,
		thingID, "name", tdJSON, "")
	if stat.Error != "" {
		return errors.New(stat.Error)
	}
	return nil
}
