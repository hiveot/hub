package service

import (
	"github.com/hiveot/hivekit/go/api/vocab"
	"github.com/hiveot/hivekit/go/api/wot"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *wot.TD {
	tdi := wot.NewTD(svc.config.ThingID, "IPNet binding", vocab.DeviceTypeService)

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
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *wot.TD {
	thingID := "urn:" + deviceInfo.MAC
	deviceName := deviceInfo.GetDefaultName()
	tdoc := wot.NewTD(thingID, deviceName, vocab.DeviceNet)

	// these are configured through the configuration file.
	prop := tdoc.AddPropertyAsString(wot.WoTTitle, "Device name", "").
		SetAtType(vocab.PropDeviceTitle)
	prop.ReadOnly = true // TODO: allow edit and save the new device name
	prop = tdoc.AddProperty(vocab.PropNetPort, "Ports", "", wot.DataTypeArray).
		SetAtType(vocab.PropNetPort)
	_ = prop
	prop = tdoc.AddPropertyAsString(vocab.PropNetHostname, "Hostname", "").
		SetAtType(vocab.PropNetHostname)
	_ = prop
	prop = tdoc.AddPropertyAsString(vocab.PropNetIP4, "IP4 address", "").
		SetAtType(vocab.PropNetIP4)
	_ = prop
	prop = tdoc.AddPropertyAsString(vocab.PropNetIP6, "IP6 address", "").
		SetAtType(vocab.PropNetIP6)
	_ = prop
	prop = tdoc.AddPropertyAsString(vocab.PropNetMAC, "MAC address", "").
		SetAtType(vocab.PropNetMAC)
	_ = prop
	prop = tdoc.AddPropertyAsInt(vocab.PropNetLatency, "Latency", "").
		SetAtType(vocab.PropNetLatency)
	prop.Unit = vocab.UnitMilliSecond

	return tdoc
}

func (svc *IPNetBinding) PubDeviceTD(deviceInfo *IPDeviceInfo) {
	tdoc := svc.MakeDeviceTD(deviceInfo)
	svc.WriteTD(tdoc)
}
