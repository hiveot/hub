package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// CreateHiveoviewTD creates a new Thing TD document describing the service capability
func (svc *HiveoviewService) CreateHiveoviewTD() *td.TD {
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
func (svc *HiveoviewService) PublishServiceTD() error {

	myTD := svc.CreateHiveoviewTD()
	tdJSON, _ := jsoniter.MarshalToString(myTD)
	err := digitwin.ThingDirectoryUpdateTD(svc.ag.Consumer, tdJSON)
	//err := svc.ag.PubTD(myTD)

	if err != nil {
		slog.Error("failed to publish the hiveoview service TD", "err", err.Error())
	}
	return err
}

// PublishServiceProps publishes the service properties
func (svc *HiveoviewService) PublishServiceProps() error {
	err := svc.ag.PubProperty(src.HiveoviewServiceID, vocab.PropNetPort, svc.port)
	if err != nil {
		slog.Error("failed to publish the hiveoview service properties", "err", err.Error())
	}
	return err
}
