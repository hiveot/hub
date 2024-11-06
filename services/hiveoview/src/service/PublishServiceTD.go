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
	//TODO: add properties or events for : uptime, nr connections, nr clients, etc

	td.AddEvent(src.NrActiveSessionsEvent, "",
		"Nr Sessions", "Number of active sessions",
		&tdd.DataSchema{
			//AtType: vocab.SessionCount,
			Type: vocab.WoTDataTypeInteger,
		})

	td.AddProperty(vocab.PropNetPort, vocab.PropNetPort,
		"UI Port", vocab.WoTDataTypeInteger)

	return td
}

// PublishServiceTD create and publish the TD of the hiveoview service
func (svc *HiveovService) PublishServiceTD() error {

	myTD := svc.CreateHiveoviewTD()
	tdJSON, _ := json.Marshal(myTD)
	err := svc.hc.PubTD(myTD.ID, string(tdJSON))

	if err != nil {
		slog.Error("failed to publish the hiveoview service TD", "err", err.Error())
	}
	return err
}

// PublishServiceProps publishes the service properties
func (svc *HiveovService) PublishServiceProps() error {
	err := svc.hc.PubProperty(src.HiveoviewServiceID, vocab.PropNetPort, svc.port)

	if err != nil {
		slog.Error("failed to publish the hiveoview service properties", "err", err.Error())
	}
	return err
}
