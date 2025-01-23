package service

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// CreateHiveoviewTD creates a new Thing TD document describing the service capability
func (svc *HiveovService) CreateHiveoviewTD() *td.TD {
	title := "Web Server"
	deviceType := vocab.ThingService
	tdi := td.NewTD(src.HiveoviewServiceID, title, deviceType)
	//TODO: add properties or events for : uptime, nr connections, nr clients, etc

	tdi.AddEvent(src.NrActiveSessionsEvent,
		"Nr Sessions", "Number of active sessions",
		&td.DataSchema{
			//AtType: vocab.SessionCount,
			Type: vocab.WoTDataTypeInteger,
		})

	tdi.AddProperty(vocab.PropNetPort, vocab.PropNetPort,
		"UI Port", vocab.WoTDataTypeInteger)

	return tdi
}

// PublishServiceTD create and publish the TD of the hiveoview service
func (svc *HiveovService) PublishServiceTD() error {

	myTD := svc.CreateHiveoviewTD()
	tdJSON, _ := json.Marshal(myTD)
	//err := svc.hc.PubTD(myTD.ID, string(tdJSON))
	notif := transports.NewNotificationResponse(wot.HTOpUpdateTD, myTD.ID, "", string(tdJSON))
	err := svc.hc.SendNotification(notif)

	if err != nil {
		slog.Error("failed to publish the hiveoview service TD", "err", err.Error())
	}
	return err
}

// PublishServiceProps publishes the service properties
func (svc *HiveovService) PublishServiceProps() error {
	notif := transports.NewNotificationResponse(
		wot.HTOpUpdateProperty, src.HiveoviewServiceID, vocab.PropNetPort, svc.port)
	err := svc.hc.SendNotification(notif)
	if err != nil {
		slog.Error("failed to publish the hiveoview service properties", "err", err.Error())
	}
	return err
}
