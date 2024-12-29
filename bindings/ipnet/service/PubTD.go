package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *td.TD {
	td := td.NewTD(svc.config.AgentID, "IPNet binding", vocab.ThingService)

	// these are configured through the configuration file.
	prop := td.AddPropertyAsInt(vocab.PropDevicePollinterval, "Poll Interval", "")
	prop.SetAtType(vocab.PropDevicePollinterval)
	prop.Unit = vocab.UnitSecond

	// nr of discovered devices is a readonly attr
	prop = td.AddPropertyAsInt("deviceCount", "Device Count", "Nr discovered devices")
	return td
}

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *td.TD {
	thingID := "urn:" + deviceInfo.MAC
	deviceName := deviceInfo.GetDefaultName()
	td := td.NewTD(thingID, deviceName, vocab.ThingNet)

	// these are configured through the configuration file.
	prop := td.AddPropertyAsString(vocab.PropDeviceTitle, "Device name", "").
		SetAtType(vocab.PropDeviceTitle)
	prop.ReadOnly = true // TODO: allow edit and save the new device name
	prop = td.AddProperty(vocab.PropNetPort, "Ports", "", vocab.WoTDataTypeArray).
		SetAtType(vocab.PropNetPort)
	prop = td.AddPropertyAsString(vocab.PropNetHostname, "Hostname", "").
		SetAtType(vocab.PropNetHostname)
	prop = td.AddPropertyAsString(vocab.PropNetIP4, "IP4 address", "").
		SetAtType(vocab.PropNetIP4)
	prop = td.AddPropertyAsString(vocab.PropNetIP6, "IP6 address", "").
		SetAtType(vocab.PropNetIP6)
	prop = td.AddPropertyAsString(vocab.PropNetMAC, "MAC address", "").
		SetAtType(vocab.PropNetMAC)
	prop = td.AddPropertyAsInt(vocab.PropNetLatency, "Latency", "").
		SetAtType(vocab.PropNetLatency)
	prop.Unit = vocab.UnitMilliSecond

	return td
}

func (svc *IPNetBinding) PubBindingTD() error {
	td := svc.MakeBindingTD()
	err := svc.hc.PubTD(td)
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	}
	return err
}

func (svc *IPNetBinding) PubDeviceTD(deviceInfo *IPDeviceInfo) error {
	td := svc.MakeDeviceTD(deviceInfo)
	err := svc.hc.PubTD(td)
	if err != nil {
		slog.Error("failed publishing device TD. Continuing...",
			slog.String("deviceID", deviceInfo.IP4),
			slog.String("err", err.Error()))
	}
	return err
}
