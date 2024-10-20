package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
)

// PublishNodeTDs reads and publishes the TD document of the gateway and its nodes.
func (svc *IsyBinding) PublishNodeTDs() (err error) {
	slog.Info("PublishNodeTDs Gateway and Nodes")

	td := svc.CreateBindingTD()
	tdJSON, _ := json.Marshal(td)
	err = svc.hc.PubTD(td.ID, string(tdJSON))
	if err != nil {
		err = fmt.Errorf("failed publishing thing TD: %w", err)
		slog.Error(err.Error())
		return err
	}
	if !svc.isyAPI.IsConnected() {
		return errors.New("not connected to the gateway")
	}

	td = svc.IsyGW.GetTD()
	if td != nil {
		tdJSON, _ = json.Marshal(td)
		err = svc.hc.PubTD(td.ID, string(tdJSON))
		if err != nil {
			err = fmt.Errorf("failed publishing gateway TD: %w", err)
			slog.Error(err.Error())
			return err
		}

		// read and publish the node TDs
		err = svc.IsyGW.ReadIsyThings()
		if err != nil {
			err = fmt.Errorf("failed reading ISY nodes from gateway: %w", err)
			slog.Error(err.Error())
			return err
		}
		//things := svc.IsyGW.ReadThings()
		for _, thing := range svc.IsyGW.GetIsyThings() {
			td = thing.GetTD()
			tdJSON, _ := json.Marshal(td)
			err = svc.hc.PubTD(td.ID, string(tdJSON))
			if err != nil {
				slog.Error("PollIsyTDs", "err", err)
			}
		}
	}
	return err
}
