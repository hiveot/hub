package service

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/ser"
	"github.com/hiveot/hub/lib/things"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *things.TD {
	thingID := svc.hc.ClientID()
	td := things.NewTD(thingID, "IPNet binding", vocab.ThingServiceAdapter)

	// these are configured through the configuration file.
	prop := td.AddPropertyAsInt(vocab.PropDevicePollinterval, vocab.PropDevicePollinterval, "Poll Interval")
	prop.Unit = vocab.UnitSecond

	// nr of discovered devices is a readonly attr
	prop = td.AddPropertyAsInt("deviceCount", "", "Nr discovered devices")
	return td
}
func (svc *IPNetBinding) MakeBindingProps() map[string]string {
	pv := make(map[string]string)
	pv[vocab.PropDevicePollinterval] = fmt.Sprintf("%d", svc.config.PollInterval)
	pv["deviceCount"] = fmt.Sprintf("%d", len(svc.devicesMap))
	return pv
}

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *things.TD {
	thingID := deviceInfo.MAC
	td := things.NewTD(thingID, "Network device", vocab.ThingNet)

	// these are configured through the configuration file.
	// FIXME: what is the best way to include a port list?
	prop := td.AddProperty("ports", "", "Ports", vocab.WoTDataTypeArray)
	prop = td.AddPropertyAsString("IP4", vocab.PropNetIP4, "IP4 address")
	prop = td.AddPropertyAsString("IP6", vocab.PropNetIP6, "IP6 address")
	prop = td.AddPropertyAsString("MAC", vocab.PropNetMAC, "MAC address")
	prop = td.AddPropertyAsInt("Latency", vocab.PropNetLatency, "Latency")
	prop.Unit = vocab.UnitMilliSecond

	return td
}

func (svc *IPNetBinding) MakeDeviceProps(deviceInfo *IPDeviceInfo) map[string]string {
	pv := make(map[string]string)
	portListJSON, _ := ser.JsonMarshal(deviceInfo.Ports)
	pv["ports"] = string(portListJSON)
	pv["IP4"] = deviceInfo.IP4
	pv["IP6"] = deviceInfo.IP6
	pv["MAC"] = deviceInfo.MAC
	pv["Latency"] = deviceInfo.Latency.String()
	return pv
}
