package service

import (
	"errors"
	"fmt"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// PublishTDs reads and publishes the TD document of the binding,
// gateway and its nodes.
func (svc *IsyBinding) PublishTDs() (err error) {
	slog.Info("PublishTDs Gateway and Nodes")

	tdi := svc.MakeBindingTD()
	tdJSON, _ := jsoniter.MarshalToString(tdi)
	err = digitwin.ThingDirectoryUpdateTD(&svc.ag.Consumer, tdJSON)
	//err = svc.ag.PubTD(tdi)
	if err != nil {
		err = fmt.Errorf("failed publishing binding TD: %w", err)
		slog.Error(err.Error())
		return err
	}

	if !svc.isyAPI.IsConnected() {
		return errors.New("not connected to the gateway")
	}

	err = svc.IsyGW.PubTD(svc.ag)
	if err == nil {
		// read and publish the node TDs
		err = svc.IsyGW.ReadIsyThings()
		if err != nil {
			err = fmt.Errorf("failed reading ISY nodes from gateway: %w", err)
			slog.Error(err.Error())
			return err
		}
		//things := svc.IsyGW.ReadThings()
		for _, thing := range svc.IsyGW.GetIsyThings() {
			tdi = thing.MakeTD()
			tdJSON, _ = jsoniter.MarshalToString(tdi)
			err = digitwin.ThingDirectoryUpdateTD(&svc.ag.Consumer, tdJSON)
			//err = svc.ag.PubTD(td)
			if err != nil {
				slog.Error("failed publishing Thing TD",
					"thingID", tdi.ID, "err", err.Error())
			}
		}
	}
	return err
}
