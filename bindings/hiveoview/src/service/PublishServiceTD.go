package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/hiveoviewapi"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// CreateHiveoviewTD creates a new Thing TD document describing the service capability
func (svc *HiveovService) CreateHiveoviewTD() *things.TD {
	title := "Web Server"
	deviceType := vocab.ThingService
	td := things.NewTD(hiveoviewapi.HiveoviewServiceID, title, deviceType)
	// TODO: add properties: uptime, max nr clients

	td.AddEvent("activeSessions", "", "Nr Sessions", "Number of currently active sessions",
		&things.DataSchema{
			//AtType: vocab.SessionCount,
			Type: vocab.WoTDataTypeInteger,
		})

	return td
}

// PublishServiceTD create and publish the TD of this service
func (svc *HiveovService) PublishServiceTD() error {

	myTD := svc.CreateHiveoviewTD()
	err := svc.hc.PubTD(myTD)

	if err != nil {
		slog.Error("failed to publish the hiveoview service TD", "err", err.Error())
	}
	return err
}
