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
	_ messaging.IConnection) *messaging.ResponseMessage {

	slog.Info("HandleRequest",
		slog.String("op", req.Operation),
		slog.String("thingID", req.ThingID),
		slog.String("name", req.Name),
		slog.String("payload", req.ToString(20)),
	)
	var err error

	// Custom config. Change device title.
	if req.Name == wot.WoTTitle {
		err = svc.SetTitle(req)
	}

	// This is a node update
	err = svc.WriteNode(req)

	// wait for the node value to change
	if err != nil {
		resp := req.CreateResponse(nil, err)
		return resp
	}
	// in the background poll the result a few times until the requested
	// status is reached or until timeout.
	go func() {
		newValue, hasUpdated := svc.WaitForValueUpdate(req, MaxUpdateWaitTime)
		if !hasUpdated {
			err = fmt.Errorf("Node didn't update property within %d seconds", MaxUpdateWaitTime)
		}
		// complete or fail the request
		resp := req.CreateResponse(newValue, err)
		_ = svc.ag.SendResponse(resp)

		// finally do a full refresh that sends notifications to subscribers
		_ = svc.RefreshPropertyValues(false)

	}()
	//return req.CreateRunningNotification(err)
	// nothing to return yet
	return nil
}

// SetTitle configures a new title of a thing
//
// OWServer doesn't support this so store this in the service store.
func (svc *OWServerBinding) SetTitle(req *messaging.RequestMessage) (err error) {

	valueStr := req.ToString(0)

	err = svc.customTitles.Set(req.ThingID, []byte(valueStr))
	if err != nil {
		slog.Error("HandleConfigRequest: Unable to save title", "err", err.Error())
	}
	if err == nil {
		// publish notification with the new title after returning
		go svc.ag.PubProperty(req.ThingID, wot.WoTTitle, valueStr)
	}
	return err
}

// WriteNode validates and writes the new value to the node
// in 1-wire actions and configuration is the same thing
func (svc *OWServerBinding) WriteNode(req *messaging.RequestMessage) (err error) {

	var attr eds.OneWireAttr
	valueStr := req.ToString(0)

	// This is a node update
	node, found := svc.nodes[req.ThingID]
	if !found {
		// delivery failed as the thingID doesn't exist
		err = fmt.Errorf("ID '%s' is not a known node", req.ThingID)
		return err
	}
	attr, found = node.Attr[req.Name]
	if !found {
		// delivery completed with error
		err = fmt.Errorf("node '%s' found but it doesn't have an attribute '%s'",
			req.ThingID, req.Name)
		return err
	} else if !attr.Writable {
		// delivery completed with error
		err = fmt.Errorf("node '%s' attribute '%s' is read-only",
			req.ThingID, req.Name)
		return err
	}

	// the thingID is the device identifier, eg the ROMId
	edsName := req.Name
	//if valueStr is a boolean then convert it to a number as 1-wire doesn't have booleans
	if valueStr == "true" {
		valueStr = "1"
	} else if valueStr == "false" {
		valueStr = "0"
	}

	err = svc.edsAPI.WriteNode(req.ThingID, edsName, valueStr)

	return err
}

// WaitForValueUpdate waits for the node value to be applied
func (svc *OWServerBinding) WaitForValueUpdate(
	req *messaging.RequestMessage, seconds int) (newValue string, hasUpdated bool) {

	var err error
	valueStr := req.ToString(0)
	hasUpdated = false

	for i := 0; i < seconds; i++ {
		newValue, err = svc.edsAPI.ReadNodeValue(req.ThingID, req.Name)
		if err == nil && valueStr == newValue {
			hasUpdated = true
			break
		}
		time.Sleep(time.Second * 1)
	}
	return newValue, hasUpdated
}
