package service

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
)

// CreateHiveoviewTD creates a new Thing TD document describing the service capability
func (svc *HiveovService) CreateHiveoviewTD() *tdd.TD {
	title := "Web Server"
	deviceType := vocab.ThingService
	td := tdd.NewTD(src.HiveoviewServiceID, title, deviceType)
	// TODO: add properties: uptime, max nr clients

	td.AddEvent(src.NrActiveSessionsEvent, "",
		"Nr Sessions", "Number of active sessions",
		&tdd.DataSchema{
			//AtType: vocab.SessionCount,
			Type: vocab.WoTDataTypeInteger,
		})

	return td
}

// PublishServiceTD create and publish the TD of this service
func (svc *HiveovService) PublishServiceTD() error {

	myTD := svc.CreateHiveoviewTD()
	tdJSON, _ := json.Marshal(myTD)
	err := svc.hc.PubTD(myTD.ID, string(tdJSON))

	if err != nil {
		slog.Error("failed to publish the hiveoview service TD", "err", err.Error())
	}
	return err
}
