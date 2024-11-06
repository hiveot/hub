package service

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
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

// MakeDeviceTD generates a TD document for discovered devices
func (svc *IPNetBinding) MakeDeviceTD(deviceInfo *IPDeviceInfo) *tdd.TD {
	thingID := "urn:" + deviceInfo.MAC
	deviceName := deviceInfo.GetDefaultName()
	td := tdd.NewTD(thingID, deviceName, vocab.ThingNet)

	// these are configured through the configuration file.
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

func (svc *IPNetBinding) PubBindingTD() error {
	td := svc.MakeBindingTD()
	tdJSON, _ := json.Marshal(td)
	err := svc.hc.PubTD(td.ID, string(tdJSON))
	if err != nil {
		slog.Error("failed publishing service TD. Continuing...",
			slog.String("err", err.Error()))
	}
	return err
}

func (svc *IPNetBinding) PubDeviceTD(deviceInfo *IPDeviceInfo) error {
	td := svc.MakeDeviceTD(deviceInfo)
	tdJSON, _ := json.Marshal(td)
	err := svc.hc.PubTD(td.ID, string(tdJSON))
	if err != nil {
		slog.Error("failed publishing device TD. Continuing...",
			slog.String("deviceID", deviceInfo.IP4),
			slog.String("err", err.Error()))
	}
	return err
}
