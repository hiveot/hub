// Package internal handles node input commands
package service

import (
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"time"
)

// SwitchOnOff turns lights or switch on or off. A payload '0', 'off' or 'false' turns off, otherwise it turns on
func (svc *IsyBinding) SwitchOnOff(thingID string, onOff bool) error {
	//
	//// any non-zero, false or off value is considered on
	//newValue := true
	//if onOffString == "0" || strings.ToLower(onOffString) == "off" || strings.ToLower(onOffString) == "false" {
	//	newValue = false
	//}
	//prevValue := "n/a"
	//prevOutputValue := pub.GetOutputValueByNodeHWID(input.NodeHWID, types.OutputType(input.InputType), input.Instance)
	//if prevOutputValue != nil {
	//	prevValue = prevOutputValue.Value
	//}

	//slog.Info("SwitchOnOff", "Name", , "Previous", prevValue, "newValue", newValue)

	// input.UpdateValue(onOffString)
	//node := svc.GetNodeByThingID(thingID)
	err := svc.IsyAPI.WriteOnOff(thingID, onOff)
	if err != nil {
		slog.Error("SwitchOnOff: error writing ISY", "err", err.Error())
	}
	return err
}

// HandleActionRequest for handling actions
// Currently very basic. Only switches are supported.
func (svc *IsyBinding) handleActionRequest(tv *things.ThingValue) (reply []byte, err error) {
	slog.Info("handleActionRequest",
		slog.String("thingID", tv.ThingID),
		slog.String("name", tv.Name),
		slog.String("senderID", tv.SenderID))

	// payloadStr := string(payload[:])

	// for now only support on/off
	switch tv.Name {
	case vocab.VocabSwitch:
		//adapter.UpdateOutputValue()device.UpdateSensorCommand(sensor, payloadStr)
		onOff := string(tv.Data) == "true"
		err = svc.SwitchOnOff(tv.ThingID, onOff)
	default:
		slog.Warn("handleActionRequest. Input is not a switch",
			slog.String("thingID", tv.ThingID),
			slog.String("name", tv.Name),
			slog.String("senderID", tv.SenderID))
		err = fmt.Errorf("Action is not a switch. Only switches are supported.")
	}
	// Give gateway time to update
	time.Sleep(300 * time.Millisecond)
	return nil, err
}
