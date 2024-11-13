// Package hubrouter with digital twin event routing functions
package router

import (
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
)

// HandlePublishEvent agent sends an event
//
// This re-submits the event with the digital twin's thing ID in the background,
// and stores it in the digital twin instance. Only the last event value is kept.
func (svc *DigitwinRouter) HandlePublishEvent(msg *hubclient.ThingMessage) {
	dThingID, err := svc.dtwStore.UpdateEventValue(
		msg.SenderID, msg.ThingID, msg.Name, msg.Data, msg.RequestID)
	if err != nil {
		return
	}
	// broadcast the event to subscribers of the digital twin
	go svc.cm.PublishEvent(dThingID, msg.Name, msg.Data, msg.RequestID, msg.SenderID)
	return
}

// HandleReadEvent consumer requests a digital twin thing's event value
func (svc *DigitwinRouter) HandleReadEvent(msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {
	reply, err := svc.dtwService.ValuesSvc.ReadEvent(msg.SenderID,
		digitwin.ValuesReadEventArgs{ThingID: msg.ThingID, Name: msg.Name})
	stat.Completed(msg, reply, err)
	return stat
}

// HandleReadAllEvents consumer requests all digital twin thing event values
func (svc *DigitwinRouter) HandleReadAllEvents(msg *hubclient.ThingMessage) (stat hubclient.RequestStatus) {
	reply, err := svc.dtwService.ValuesSvc.ReadAllEvents(msg.SenderID, msg.ThingID)
	stat.Completed(msg, reply, err)
	return stat
}
