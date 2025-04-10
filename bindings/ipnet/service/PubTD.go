package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
)

// MakeBindingTD generates a TD document for this binding
// containing configuration properties, event and action definitions
func (svc *IPNetBinding) MakeBindingTD() *td.TD {
	tdi := td.NewTD(svc.config.AgentID, "IPNet binding", vocab.ThingService)

	// these are configured through the configuration file.
	prop := tdi.AddPropertyAsInt(vocab.PropDevicePollinterval, "Poll Interval", "")
	prop.SetAtType(vocab.PropDevicePollinterval)
	prop.Unit = vocab.UnitSecond

	// nr of discovered devices is a readonly attr
	prop = tdi.AddPropertyAsInt("deviceCount", "Device Count", "Nr discovered devices")
	return tdi
}

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *td.TD {
	thingID := "urn:" + deviceInfo.MAC
	deviceName := deviceInfo.GetDefaultName()
	tdi := td.NewTD(thingID, deviceName, vocab.ThingNet)

	// these are configured through the configuration file.
	prop := tdi.AddPropertyAsString(wot.WoTTitle, "Device name", "").
		SetAtType(vocab.PropDeviceTitle)
	prop.ReadOnly = true // TODO: allow edit and save the new device name
	prop = tdi.AddProperty(vocab.PropNetPort, "Ports", "", vocab.WoTDataTypeArray).
		SetAtType(vocab.PropNetPort)
	prop = tdi.AddPropertyAsString(vocab.PropNetHostname, "Hostname", "").
		SetAtType(vocab.PropNetHostname)
	prop = tdi.AddPropertyAsString(vocab.PropNetIP4, "IP4 address", "").
		SetAtType(vocab.PropNetIP4)
	prop = tdi.AddPropertyAsString(vocab.PropNetIP6, "IP6 address", "").
		SetAtType(vocab.PropNetIP6)
	prop = tdi.AddPropertyAsString(vocab.PropNetMAC, "MAC address", "").
		SetAtType(vocab.PropNetMAC)
	prop = tdi.AddPropertyAsInt(vocab.PropNetLatency, "Latency", "").
		SetAtType(vocab.PropNetLatency)
	prop.Unit = vocab.UnitMilliSecond

	return tdi
}

func (svc *IPNetBinding) PubBindingTD() error {
	tdi := svc.MakeBindingTD()
	tdJSON, _ := jsoniter.MarshalToString(tdi)
	err := digitwin.ThingDirectoryUpdateTD(svc.ag.Consumer, tdJSON)
	//err := svc.ag.PubTD(tdi)
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	}
	return err
}

func (svc *IPNetBinding) PubDeviceTD(deviceInfo *IPDeviceInfo) error {
	tdi := svc.MakeDeviceTD(deviceInfo)
	tdJSON, _ := jsoniter.MarshalToString(tdi)
	err := digitwin.ThingDirectoryUpdateTD(svc.ag.Consumer, tdJSON)
	//err := svc.ag.PubTD(tdi)
	if err != nil {
		slog.Error("failed publishing device TD. Continuing...",
			slog.String("deviceID", deviceInfo.IP4),
			slog.String("err", err.Error()))
	}
	return err
}
