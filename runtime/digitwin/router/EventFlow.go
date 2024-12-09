// Package hubrouter with digital twin event routing functions
package router

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/transports"
)

// HandlePublishEvent agent sends an event for broadcast to subscribers
//
// This re-submits the event with the digital twin's thing ID in the background,
// and stores it in the digital twin instance. Only the last event value is kept.
func (svc *DigitwinRouter) HandlePublishEvent(
	msg *transports.ThingMessage) (completed bool, output any, err error) {

	dThingID, err := svc.dtwStore.UpdateEventValue(
		msg.SenderID, msg.ThingID, msg.Name, msg.Data, msg.RequestID)
	if err == nil {
		// broadcast the event to subscribers of the digital twin
		go svc.cm.PublishEvent(dThingID, msg.Name, msg.Data, msg.RequestID, msg.SenderID)
	}
	return true, nil, err
}

// HandleReadEvent consumer requests a digital twin thing's event value
func (svc *DigitwinRouter) HandleReadEvent(
	msg *transports.ThingMessage) (completed bool, output any, err error) {

	output, err = svc.dtwService.ValuesSvc.ReadEvent(msg.SenderID,
		digitwin.ValuesReadEventArgs{ThingID: msg.ThingID, Name: msg.Name})
	return true, output, err
}

// HandleReadAllEvents consumer requests all digital twin thing event values
func (svc *DigitwinRouter) HandleReadAllEvents(
	msg *transports.ThingMessage) (completed bool, output any, err error) {

	output, err = svc.dtwService.ValuesSvc.ReadAllEvents(msg.SenderID, msg.ThingID)
	return true, output, err
}
