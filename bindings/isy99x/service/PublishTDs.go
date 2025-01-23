package service

import (
	"errors"
	"fmt"
	"log/slog"
)

// PublishTDs reads and publishes the TD document of the binding,
// gateway and its nodes.
func (svc *IsyBinding) PublishTDs() (err error) {
	slog.Info("PublishTDs Gateway and Nodes")

	td := svc.MakeBindingTD()
	err = svc.ag.PubTD(td)
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
			td = thing.MakeTD()
			err = svc.ag.PubTD(td)
			if err != nil {
				slog.Error("failed publishing Thing TD",
					"thingID", td.ID, "err", err.Error())
			}
		}
	}
	return err
}
