package service

import (
	"errors"
	"fmt"
	"log/slog"
)

// PublishNodeTDs reads and publishes the TD document of the gateway and its nodes.
func (svc *IsyBinding) PublishNodeTDs() (err error) {
	slog.Info("PublishNodeTDs Gateway and Nodes")

	err = svc.PubTD()
	if err != nil {
		err = fmt.Errorf("failed publishing thing TD: %w", err)
		slog.Error(err.Error())
		return err
	}
	if !svc.isyAPI.IsConnected() {
		return errors.New("not connected to the gateway")
	}

	err = svc.IsyGW.PubTD(svc.hc)
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
			err = thing.PubTD(svc.hc)
			if err != nil {
				slog.Error("failed publishing ISY Thing TD", "err", err)
			}
		}
	}
	return err
}
