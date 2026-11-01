package service

import (
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/api/vocab"
	jsoniter "github.com/json-iterator/go"
)

func (svc *IPNetBinding) MakeDeviceProps(deviceInfo *IPDeviceInfo) map[string]string {
	pv := make(map[string]string)
	portListJSON, _ := jsoniter.Marshal(deviceInfo.Ports)
	// TODO: Use the saved device name
	pv[td.WoTTitle] = deviceInfo.GetDefaultName()
	pv[vocab.PropNetHostname] = deviceInfo.Hostname
	pv[vocab.PropNetPort] = string(portListJSON)
	pv[vocab.PropNetIP4] = deviceInfo.IP4
	pv[vocab.PropNetIP6] = deviceInfo.IP6
	pv[vocab.PropNetMAC] = deviceInfo.MAC
	pv[vocab.PropNetLatency] = deviceInfo.Latency.String()
	return pv
}

// Publish the properties of this binding
func (svc *IPNetBinding) PubBindingProps(onlyChanges bool) {
	svc.PubProperty("", vocab.PropDevicePollinterval, svc.config.PollInterval, onlyChanges)
	svc.PubProperty("", "deviceCount", len(svc.devicesMap), onlyChanges)
}
