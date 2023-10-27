// Package internal handles node input commands
package internal

import (
	"strings"
	"time"

	"github.com/iotdomain/iotdomain-go/types"
	"github.com/sirupsen/logrus"
)

// SwitchOnOff turns lights or switch on or off. A payload '0', 'off' or 'false' turns off, otherwise it turns on
func (app *IsyApp) SwitchOnOff(input *types.InputDiscoveryMessage, onOffString string) error {
	pub := app.pub
	// any non-zero, false or off value is considered on
	newValue := true
	if onOffString == "0" || strings.ToLower(onOffString) == "off" || strings.ToLower(onOffString) == "false" {
		newValue = false
	}
	prevValue := "n/a"
	prevOutputValue := pub.GetOutputValueByNodeHWID(input.NodeHWID, types.OutputType(input.InputType), input.Instance)
	if prevOutputValue != nil {
		prevValue = prevOutputValue.Value
	}

	logrus.Infof("IsyApp.SwitchOnOff: Address %s. Previous value=%s, New value=%v", input.Address, prevValue, newValue)

	// input.UpdateValue(onOffString)
	node := pub.GetNodeByAddress(input.Address)
	err := app.isyAPI.WriteOnOff(node.HWID, newValue)
	if err != nil {
		logrus.Errorf("IsyApp.SwitchOnOff: Input %s: error writing ISY: %v", input.Address, err)
	}
	return err
}

// HandleInputCommand for handling input commands
// Currently very basic. Only switches are supported.
func (app *IsyApp) HandleInputCommand(
	input *types.InputDiscoveryMessage, sender string, value string) {
	logrus.Infof("IsyApp.HandleInputCommand. Input for '%s'", input.Address)

	// payloadStr := string(payload[:])

	// for now only support on/off
	switch input.InputType {
	case types.InputTypeSwitch:
		//adapter.UpdateOutputValue()device.UpdateSensorCommand(sensor, payloadStr)
		_ = app.SwitchOnOff(input, value)
	default:
		logrus.Warningf("IsyApp.HandleInputCommand. Input '%s' is Not a switch", input.Address)
	}
	// publish the result. give gateway time to update.
	// TODO: get push notification instead
	// Give gateway time to update.
	time.Sleep(300 * time.Millisecond)
	app.Poll(app.pub)
}
