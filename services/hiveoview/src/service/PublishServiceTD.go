package service

import (
	"fmt"
	"log/slog"

	"github.com/hiveot/hivekit/go/utils/net"
	"github.com/hiveot/hivekit/go/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/services/hiveoview/src"
	jsoniter "github.com/json-iterator/go"
)

// CreateServiceTD creates a new Thing TD document for this service
func (svc *HiveoviewService) CreateServiceTD() *td.TD {
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
	myAddress := fmt.Sprintf("%s:%d", net.GetOutboundIP("").String(), svc.port)
	prop := tdi.AddProperty(vocab.PropNetPort, "Server Listening Port",
		fmt.Sprintf("Web server listening port (for example: %d in https://%s)", svc.port, myAddress),
		vocab.WoTDataTypeInteger)
	prop.AtType = vocab.PropNetPort

	return tdi
}

// PublishServiceTD create and publish the TD of the hiveoview service
func (svc *HiveoviewService) PublishServiceTD() error {

	myTD := svc.CreateServiceTD()
	tdJSON, _ := jsoniter.MarshalToString(myTD)
	err := digitwin.ThingDirectoryUpdateThing(svc.ag.Consumer, tdJSON)
	//err := svc.ag.UpdateThing(myTD)

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
