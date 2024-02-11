package service

import (
	"fmt"
	"log/slog"
)

// PollGatewayNodes reads and publishes a TD of the gateway and nodes
func (svc *IsyBinding) PollGatewayNodes() error {
	slog.Info("Polling Gateway and Nodes")
	// poll and publish the ISY gateway device
	isyGW, err := svc.IsyAPI.ReadIsyGateway()
	isyNodes, err2 := svc.IsyAPI.ReadIsyNodes()
	if err2 != nil {
		err = err2
	}
	if err != nil {
		err = fmt.Errorf("failed reading ISY: %w", err)
		slog.Error(err.Error())
		return err
	}

	td := svc.MakeGatewayTD(isyGW)
	err = svc.hc.PubTD(td)
	// no use continuing if publishing fails
	if err != nil {
		err = fmt.Errorf("failed publishing gateway TD: %w", err)
		slog.Error(err.Error())
		return err
	} else {
		props := svc.MakeGatewayProps(isyGW)
		_ = svc.hc.PubProps(td.ID, props)
	}

	for _, node := range isyNodes {
		td := svc.MakeNodeTD(node)
		err = svc.hc.PubTD(td)
		// no use continuing if publishing fails
		if err != nil {
			err = fmt.Errorf("failed publishing node TDs: %w", err)
			slog.Error(err.Error())
			return err
		} else {
			props := svc.MakeNodeProps(node)
			_ = svc.hc.PubProps(td.ID, props)
		}
	}
	return err
}

// PollValues reads and publishes node values
// Set onlyChanges to only publish changed values
func (svc *IsyBinding) PollValues(onlyChanges bool) error {
	slog.Info("Polling Values", slog.Bool("onlyChanges", onlyChanges))
	isyStatus, err := svc.IsyAPI.ReadIsyStatus()
	if err != nil {
		err = fmt.Errorf("failed reading ISY: %w", err)
		slog.Error(err.Error())
		return err
	}

	// send each value as an event, if changed
	for _, nodeStatus := range isyStatus.Nodes {
		key := nodeStatus.Address + "/" + nodeStatus.Prop.ID
		prevValue, found := svc.prevValues[key]
		if !found || !onlyChanges || prevValue != nodeStatus.Prop.Value {
			svc.prevValues[key] = nodeStatus.Prop.Value
			valueName := svc.getPropName(&nodeStatus.Prop)
			err = svc.hc.PubEvent(
				nodeStatus.Address,
				valueName,
				[]byte(nodeStatus.Prop.Value))
		}
		// no use continuing if publishing fails
		if err != nil {
			err = fmt.Errorf("failed publishing ISY event: %w", err)
			slog.Error(err.Error())
			return err
		}
	}
	return nil
}
