package service

import (
	"encoding/json"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"time"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *things.TD {
	thingID := svc.hc.ClientID()
	td := things.NewTD(thingID, "IPNet binding", vocab.DeviceTypeBinding)

	// these are configured through the configuration file.
	prop := td.AddPropertyAsInt(vocab.VocabPollInterval, vocab.VocabPollInterval, "Poll Interval", 3600)
	prop.Unit = vocab.UnitNameSecond

	// nr of discovered devices is a readonly attr
	count := len(svc.devicesMap)
	prop = td.AddPropertyAsInt("deviceCount", "", "Nr discovered devices", count)
	return td
}

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *things.TD {
	thingID := deviceInfo.MAC
	td := things.NewTD(thingID, "Network device", vocab.DeviceTypeNetwork)

	// these are configured through the configuration file.
	portList, _ := json.Marshal(deviceInfo.Ports)
	// FIXME: what is the best way to include a port list?
	prop := td.AddProperty("ports", "", "Ports", vocab.WoTDataTypeString, string(portList))
	prop = td.AddPropertyAsString("IP4", vocab.VocabLocalIP, "IP4 address", deviceInfo.IP4)
	prop = td.AddPropertyAsString("IP6", vocab.VocabLocalIP, "IP6 address", deviceInfo.IP6)
	prop = td.AddPropertyAsString("MAC", vocab.VocabMAC, "MAC address", deviceInfo.MAC)
	prop = td.AddPropertyAsInt("Latenct", vocab.VocabLatency, "Latency", int(deviceInfo.Latency/time.Millisecond))
	prop.Unit = vocab.UnitNameMilliSecond

	return td
}
