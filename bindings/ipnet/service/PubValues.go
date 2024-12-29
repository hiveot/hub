package service

import (
	"github.com/hiveot/hub/api/go/vocab"
	jsoniter "github.com/json-iterator/go"
)

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

func (svc *IPNetBinding) PubBindingProps() {
	thingID := svc.config.AgentID
	_ = svc.hc.PubProperty(thingID, vocab.PropDevicePollinterval, svc.config.PollInterval)
	_ = svc.hc.PubProperty(thingID, "deviceCount", len(svc.devicesMap))
}
