package httpbinding

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/transports"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// Pub an action, event, property, td or progress message and return the delivery status
//
//	methodName is http.MethodPost for actions, http.MethodPost/MethodGet for properties
//	path used to publish PostActionPath/PostEventPath/... optionally with {thingID} and/or {name}
//	thingID (optional) to publish as or to: events are published for the thing and actions to publish to the thingID
//	name (optional) is the event/action/property name being published or modified
//	input is the native message payload to transfer that will be serialized
//	output is optional destination for unmarshalling the payload
//	requestID optional 'message-id' header value
//
// This returns the response body and optional a response message with delivery status and requestID with a delivery status
func (cl *HttpBindingClient) Pub(methodName string, methodPath string,
	thingID string, name string, input interface{}, correlationID string) (
	stat transports.RequestStatus) {

	var data []byte
	progress := ""

	vars := map[string]string{
		"thingID": thingID,
		"name":    name}
	messagePath := utils.Substitute(methodPath, vars)
	if input != nil {
		data, _ = jsoniter.Marshal(input)
	}
	resp, headers, err := cl.Invoke(methodName, messagePath, "", data, nil)

	// TODO: detect difference between not connected and unauthenticated
	dataSchema := ""
	// parse response headers
	if headers != nil {
		// set if an alternative output dataschema is used, eg RequestStatus result
		dataSchema = headers.Get(DataSchemaHeader)
		// when progress is returned without a deliverystatus object
		progress = headers.Get(StatusHeader)
	}

	stat.CorrelationID = correlationID
	if err != nil {
		stat.Error = err.Error()
		stat.Status = vocab.RequestFailed
	} else if dataSchema == "RequestStatus" {
		// return dataschema contains a progress status envelope
		err = jsoniter.Unmarshal(resp, &stat)
	} else if resp != nil && len(resp) > 0 {
		// TODO: unmarshalling the reply here is useless as there is needs conversion to the correct type
		err = jsoniter.Unmarshal(resp, &stat.Output)
		stat.Status = vocab.RequestCompleted
	} else if progress != "" {
		// progress status without delivery status output
		stat.Status = progress
	} else {
		// not an progress result and no data. assume all went well
		stat.Status = vocab.RequestCompleted
	}
	if err != nil {
		slog.Error("Pub error",
			"path", messagePath, "err", err.Error())
		stat.Error = err.Error()
	}
	return stat
}

//
//// WriteProperty posts a configuration change request
//func (cl *HttpBindingClient) WriteProperty(thingID string, name string, data any) (
//	stat transports.RequestStatus) {
//
//	slog.Info("WriteProperty",
//		slog.String("me", cl.clientID),
//		slog.String("thingID", thingID),
//		slog.String("name", name),
//	)
//
//	stat = cl.Pub(http.MethodPost, PostWritePropertyPath, thingID, name, data, "")
//	return stat
//}
