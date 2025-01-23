// Package internal handles input set command
package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"log/slog"
	"time"
)

// HandleRequest handles action or property write requests
func (svc *OWServerBinding) HandleRequest(req *transports.RequestMessage,
	_ transports.IConnection) (resp *transports.ResponseMessage) {

	slog.Info("HandleRequest",
		slog.String("op", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("payload", req.ToString(20)),
	)

	if req.Operation == wot.OpWriteProperty {
		return svc.HandleConfigRequest(req)
	} else if req.Operation == wot.OpInvokeAction {
		return svc.HandleActionRequest(req)
	}
	err := fmt.Errorf("Unknown operation '%s'", req.Operation)
	resp = req.CreateResponse(nil, err)
	slog.Warn("HandleRequest failed", "err", resp.Error)
	return resp
}

// HandleActionRequest handles requests to activate inputs
func (svc *OWServerBinding) HandleActionRequest(req *transports.RequestMessage) (resp *transports.ResponseMessage) {
	var attr eds.OneWireAttr

	// TODO: lookup the req Title used by the EDS
	edsName := req.Name

	node, found := svc.nodes[req.ThingID]
	if !found {
		// delivery failed as the thingID doesn't exist
		err := fmt.Errorf("ID '%s' is not a known node", req.ThingID)
		resp = req.CreateResponse(nil, err)
		return resp
	}
	attr, found = node.Attr[req.Name]
	if !found {
		// delivery completed with error
		err := fmt.Errorf("node '%s' found but it doesn't have an req '%s'",
			req.ThingID, req.Name)
		resp = req.CreateResponse(nil, err)
		return resp
	} else if !attr.Writable {
		// delivery completed with error
		err := fmt.Errorf("node '%s' req '%s' is a read-only attribute",
			req.ThingID, req.Name)
		resp = req.CreateResponse(nil, err)
		return resp
	}

	// Determine the value.
	// FIXME: when building the TD, Booleans are defined as enum integers
	actionValue := req.ToString(0)
	var err error

	// the thingID is the device identifier, eg the ROMId
	err = svc.edsAPI.WriteData(req.ThingID, edsName, string(actionValue))

	// read the result
	time.Sleep(time.Second)
	_ = svc.RefreshPropertyValues(false)

	// Writing the EDS is slow, retry in case it was missed
	time.Sleep(time.Second * 1)
	_ = svc.RefreshPropertyValues(false)

	if err != nil {
		err = fmt.Errorf("req '%s' failed: %w", req.Name, err)
	}

	resp = req.CreateResponse(nil, err)
	return resp
}

// HandleConfigRequest handles requests to configure the service or devices
func (svc *OWServerBinding) HandleConfigRequest(req *transports.RequestMessage) (stat *transports.ResponseMessage) {
	var err error
	valueStr := req.ToString(0)
	slog.Info("HandleConfigRequest",
		slog.String("thingID", req.ThingID),
		slog.String("property", req.Name),
		slog.String("payload", req.ToString(20)))

	// the thingID is the ROMId of the device to configure
	// the Name is the attributeID of the property to configure
	node, found := svc.nodes[req.ThingID]
	if !found {
		// unable to delivery to Thing
		err := fmt.Errorf("HandleConfigRequest: Thing '%s' not found", req.ThingID)
		slog.Warn(err.Error())
		return req.CreateResponse(nil, err)
	}

	// custom config. Configure the device title and save it in the state service.
	if req.Name == vocab.PropDeviceTitle {
		svc.customTitles[req.ThingID] = valueStr
		go svc.SaveState()
		// publish changed values after returning
		go svc.ag.PubProperty(req.ThingID, vocab.PropDeviceTitle, valueStr)
		return req.CreateResponse(valueStr, nil)
	} else {
		attr, found := node.Attr[req.Name]
		if !found {
			err = fmt.Errorf("HandleConfigRequest: '%s not a property of Thing '%s' found",
				req.Name, req.ThingID)
			slog.Warn(err.Error())
		} else if !attr.Writable {
			err := fmt.Errorf(
				"HandleConfigRequest: property '%s' of Thing '%s' is not writable",
				req.Name, req.ThingID)
			slog.Warn(err.Error())
		} else {
			err = svc.edsAPI.WriteData(req.ThingID, req.Name, valueStr)
			// there is no output in the response as this can take some time
		}
		// publish changed value after returning
		go func() {
			// owserver is slow to update
			// FIXME: track config updates
			time.Sleep(time.Second * 1)
			_ = svc.RefreshPropertyValues(false)
		}()
	}

	return req.CreateResponse(nil, err)
}
