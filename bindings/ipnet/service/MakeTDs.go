package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *tdd.TD {
	td := tdd.NewTD(svc.config.AgentID, "IPNet binding", vocab.ThingService)

	// these are configured through the configuration file.
	prop := td.AddPropertyAsInt(vocab.PropDevicePollinterval, vocab.PropDevicePollinterval, "Poll Interval")
	prop.Unit = vocab.UnitSecond

	// nr of discovered devices is a readonly attr
	prop = td.AddPropertyAsInt("deviceCount", "", "Nr discovered devices")
	return td
}
func (svc *IPNetBinding) MakeBindingProps() map[string]any {
	pv := make(map[string]any)
	pv[vocab.PropDevicePollinterval] = svc.config.PollInterval
	pv["deviceCount"] = fmt.Sprintf("%d", len(svc.devicesMap))
	return pv
}

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *tdd.TD {
	thingID := "urn:" + deviceInfo.MAC
	deviceName := deviceInfo.GetDefaultName()
	td := tdd.NewTD(thingID, deviceName, vocab.ThingNet)

	// these are configured through the configuration file.
	// FIXME: what is the best way to include a port list?
	prop := td.AddPropertyAsString("", vocab.PropDeviceTitle, "Device name")
	prop.ReadOnly = true // TODO: allow edit and save the new device name
	prop = td.AddProperty("", vocab.PropNetPort, "Ports", vocab.WoTDataTypeArray)
	prop = td.AddPropertyAsString("", vocab.PropNetHostname, "Hostname")
	prop = td.AddPropertyAsString("", vocab.PropNetIP4, "IP4 address")
	prop = td.AddPropertyAsString("", vocab.PropNetIP6, "IP6 address")
	prop = td.AddPropertyAsString("", vocab.PropNetMAC, "MAC address")
	prop = td.AddPropertyAsInt("", vocab.PropNetLatency, "Latency")
	prop.Unit = vocab.UnitMilliSecond

	return td
}

func (svc *IPNetBinding) MakeDeviceProps(deviceInfo *IPDeviceInfo) map[string]string {
	pv := make(map[string]string)
	portListJSON, _ := jsoniter.Marshal(deviceInfo.Ports)
	// TODO: Use the saved device name
	pv[vocab.PropDeviceTitle] = deviceInfo.GetDefaultName()
	pv[vocab.PropNetHostname] = deviceInfo.Hostname
	pv[vocab.PropNetPort] = string(portListJSON)
	pv[vocab.PropNetIP4] = deviceInfo.IP4
	pv[vocab.PropNetIP6] = deviceInfo.IP6
	pv[vocab.PropNetMAC] = deviceInfo.MAC
	pv[vocab.PropNetLatency] = deviceInfo.Latency.String()
	return pv
}
