package service

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *td.TD {
	tdi := td.NewTD(svc.config.ThingID, "IPNet binding", vocab.DeviceTypeService)

	// these are configured through the configuration file.
	prop := tdi.AddPropertyAsInt(vocab.PropDevicePollinterval, "Poll Interval", "")
	prop.SetAtType(vocab.PropDevicePollinterval)
	prop.Unit = vocab.UnitSecond

	// nr of discovered devices is a readonly attr
	prop = tdi.AddPropertyAsInt("deviceCount", "Device Count", "Nr discovered devices")
	_ = prop
	return tdi
}

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *td.TD {
	thingID := "urn:" + deviceInfo.MAC
	deviceName := deviceInfo.GetDefaultName()
	tdi := td.NewTD(thingID, deviceName, vocab.DeviceNet)

	// these are configured through the configuration file.
	prop := tdi.AddPropertyAsString(td.WoTTitle, "Device name", "").
		SetAtType(vocab.PropDeviceTitle)
	prop.ReadOnly = true // TODO: allow edit and save the new device name
	prop = tdi.AddProperty(vocab.PropNetPort, "Ports", "", td.DataTypeArray).
		SetAtType(vocab.PropNetPort)
	_ = prop
	prop = tdi.AddPropertyAsString(vocab.PropNetHostname, "Hostname", "").
		SetAtType(vocab.PropNetHostname)
	_ = prop
	prop = tdi.AddPropertyAsString(vocab.PropNetIP4, "IP4 address", "").
		SetAtType(vocab.PropNetIP4)
	_ = prop
	prop = tdi.AddPropertyAsString(vocab.PropNetIP6, "IP6 address", "").
		SetAtType(vocab.PropNetIP6)
	_ = prop
	prop = tdi.AddPropertyAsString(vocab.PropNetMAC, "MAC address", "").
		SetAtType(vocab.PropNetMAC)
	_ = prop
	prop = tdi.AddPropertyAsInt(vocab.PropNetLatency, "Latency", "").
		SetAtType(vocab.PropNetLatency)
	prop.Unit = vocab.UnitMilliSecond

	return tdi
}

func (svc *IPNetBinding) PubDeviceTD(deviceInfo *IPDeviceInfo) {
	tdoc := svc.MakeDeviceTD(deviceInfo)
	svc.WriteTD(tdoc)
}
