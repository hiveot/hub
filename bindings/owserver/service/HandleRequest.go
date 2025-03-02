// Package internal handles input set command
package service

import (
	"fmt"
	"github.com/hiveot/hub/bindings/owserver/service/eds"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot"
	"log/slog"
	"time"
)

const MaxUpdateWaitTime = 5

// HandleRequest handles action or property write requests
// For 1-wire, configuration and actions are the same thing
func (svc *OWServerBinding) HandleRequest(req *messaging.RequestMessage,
	_ messaging.IConnection) (resp *messaging.ResponseMessage) {

	slog.Info("HandleRequest",
		slog.String("op", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("payload", req.ToString(20)),
	)

	var attr eds.OneWireAttr
	var err error

	valueStr := req.ToString(0)

	// custom config. Configure the device title and save it in the state service.
	// TODO setting title will move to the digital twin
	if req.Name == wot.WoTTitle {
		err = svc.customTitles.Set(req.ThingID, []byte(valueStr))
		if err != nil {
			slog.Error("HandleConfigRequest: Unable to save title", "err", err.Error())
		}
		if err == nil {
			// publish changed values after returning
			go svc.ag.PubProperty(req.ThingID, wot.WoTTitle, valueStr)
		}
		// completed
		return req.CreateResponse(valueStr, err)
	}

	// This is a node update
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
		err := fmt.Errorf("node '%s' found but it doesn't have an attribute '%s'",
			req.ThingID, req.Name)
		resp = req.CreateResponse(nil, err)
		return resp
	} else if !attr.Writable {
		// delivery completed with error
		err := fmt.Errorf("node '%s' attribute '%s' is read-only",
			req.ThingID, req.Name)
		resp = req.CreateResponse(nil, err)
		return resp
	}

	// FIXME: when building the TD, Booleans are defined as enum integers

	// the thingID is the device identifier, eg the ROMId
	edsName := req.Name
	err = svc.edsAPI.WriteNode(req.ThingID, edsName, valueStr)

	if err == nil {
		var newValue string

		// in the background poll the result a few times until the requested
		// status is reached or until timeout.
		go func() {
			hasUpdated := false
			for i := 0; i < MaxUpdateWaitTime; i++ {
				newValue, err = svc.edsAPI.ReadNodeValue(req.ThingID, req.Name)
				if err == nil && valueStr == newValue {
					hasUpdated = true
					break
				}
				time.Sleep(time.Second * 1)
			}
			if !hasUpdated {
				err = fmt.Errorf("Node didn't update property within %d seconds", MaxUpdateWaitTime)
			}
			// complete or fail the request
			resp = req.CreateResponse(newValue, err)
			_ = svc.ag.SendResponse(resp)
			// finally do a full refresh that sends notifications to subscribers
			_ = svc.RefreshPropertyValues(false)
		}()
	}
	return req.CreateRunningResponse(err)
}
