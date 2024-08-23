package service

import (
	"fmt"
	vocab2 "github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/wot/tdd"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *tdd.TD {
	td := tdd.NewTD(svc.config.AgentID, "IPNet binding", vocab2.ThingServiceAdapter)

	// these are configured through the configuration file.
	prop := td.AddPropertyAsInt(vocab2.PropDevicePollinterval, vocab2.PropDevicePollinterval, "Poll Interval")
	prop.Unit = vocab2.UnitSecond

	// nr of discovered devices is a readonly attr
	prop = td.AddPropertyAsInt("deviceCount", "", "Nr discovered devices")
	return td
}
func (svc *IPNetBinding) MakeBindingProps() map[string]any {
	pv := make(map[string]any)
	pv[vocab2.PropDevicePollinterval] = svc.config.PollInterval
	pv["deviceCount"] = fmt.Sprintf("%d", len(svc.devicesMap))
	return pv
}

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *tdd.TD {
	thingID := "urn:" + deviceInfo.MAC
	deviceName := deviceInfo.GetDefaultName()
	td := tdd.NewTD(thingID, deviceName, vocab2.ThingNet)

	// these are configured through the configuration file.
	// FIXME: what is the best way to include a port list?
	prop := td.AddPropertyAsString("", vocab2.PropDeviceTitle, "Device name")
	prop.ReadOnly = true // TODO: allow edit and save the new device name
	prop = td.AddProperty("", vocab2.PropNetPort, "Ports", vocab2.WoTDataTypeArray)
	prop = td.AddPropertyAsString("", vocab2.PropNetHostname, "Hostname")
	prop = td.AddPropertyAsString("", vocab2.PropNetIP4, "IP4 address")
	prop = td.AddPropertyAsString("", vocab2.PropNetIP6, "IP6 address")
	prop = td.AddPropertyAsString("", vocab2.PropNetMAC, "MAC address")
	prop = td.AddPropertyAsInt("", vocab2.PropNetLatency, "Latency")
	prop.Unit = vocab2.UnitMilliSecond

	return td
}

func (svc *IPNetBinding) MakeDeviceProps(deviceInfo *IPDeviceInfo) map[string]string {
	pv := make(map[string]string)
	portListJSON, _ := ser.JsonMarshal(deviceInfo.Ports)
	// TODO: Use the saved device name
	pv[vocab2.PropDeviceTitle] = deviceInfo.GetDefaultName()
	pv[vocab2.PropNetHostname] = deviceInfo.Hostname
	pv[vocab2.PropNetPort] = string(portListJSON)
	pv[vocab2.PropNetIP4] = deviceInfo.IP4
	pv[vocab2.PropNetIP6] = deviceInfo.IP6
	pv[vocab2.PropNetMAC] = deviceInfo.MAC
	pv[vocab2.PropNetLatency] = deviceInfo.Latency.String()
	return pv
}
