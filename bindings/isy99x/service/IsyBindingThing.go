// Package service for basic ISY99x Insteon home automation hub access
// This implements common sensors and switches
package service

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
)

// GetPropValues returns the property/event values to publish
func (svc *IsyBinding) GetPropValues(onlyChanges bool) (map[string]string, map[string]string) {
	props := make(map[string]string)
	props[vocab.PropDevicePollinterval] = fmt.Sprintf("%d", svc.config.PollInterval)
	props[vocab.PropNetAddress] = svc.config.IsyAddress
	props[vocab.PropDeviceMake] = "Hive Of Things"

	connStatus := "disconnnected"
	if svc.isyAPI.IsConnected() {
		connStatus = "connected"
	}
	events := make(map[string]string)
	events[vocab.PropNetConnection] = connStatus
	//

	return props, events
}

// GetTD generates a TD document for this binding containing properties,
// event and action definitions.
func (svc *IsyBinding) GetTD() *things.TD {
	td := things.NewTD(svc.thingID, "ISY99x binding", vocab.ThingServiceAdapter)

	// binding attributes
	prop := td.AddProperty("connectionStatus", "",
		"Connection Status", vocab.WoTDataTypeString)
	prop.Description = "Whether the Binding has a connection to an ISY gateway"
	//
	prop = td.AddProperty(vocab.PropDeviceMake, vocab.PropDeviceMake,
		"Manufacturer", vocab.WoTDataTypeString)
	prop.Description = "Developer of the binding"

	// TODO: persist configuration
	//binding config
	prop = td.AddProperty(vocab.PropDevicePollinterval, vocab.PropDevicePollinterval,
		"Poll Interval", vocab.WoTDataTypeInteger)
	prop.Description = "Interval the binding polls the gateway for data value updates."
	prop.Unit = vocab.UnitSecond
	prop.ReadOnly = false
	//
	prop = td.AddProperty(vocab.PropNetAddress, vocab.PropNetAddress,
		"Network Address", vocab.WoTDataTypeString)
	prop.Description = "ISY99x IP address; empty to auto discover."
	prop.ReadOnly = false

	// binding events
	ev := td.AddEvent("", vocab.PropNetConnection, "Connection status", vocab.WoTDataTypeNone, nil)
	ev.Description = "Status of connection to OWServer gateway changed"

	// no binding actions
	return td
}

// HandleBindingConfig configures the binding.
func (svc *IsyBinding) HandleBindingConfig(tv *things.ThingValue) error {
	err := fmt.Errorf("unknown configuration request '%s' from '%s'", tv.Name, tv.SenderID)
	switch tv.Name {
	case vocab.PropNetAddress:
		svc.config.IsyAddress = string(tv.Data)
		err = svc.isyAPI.Connect(svc.config.IsyAddress, svc.config.LoginName, svc.config.Password)
		if err == nil {
			svc.IsyGW.Init(svc.isyAPI)
		}
	}
	return err
}

//// GetValues returns the property values of the binding Thing
//func (svc *IsyBinding) GetValues(onlyChanges bool) map[string]string {
//	props := make(map[string]string)
//	props[vocab.VocabPollInterval] = fmt.Sprintf("%d", svc.config.PollInterval)
//	props[vocab.VocabGatewayAddress] = svc.config.IsyAddress
//	return props
//}

// Run the publisher until the SIGTERM  or SIGINT signal is received
//func Run() error {
//	appConfig := &IsyBindingConfig{ClientID: appID}
//	hc := hubclient.NewHubClient("", caCert, core)
//	err := hc.ConnectWithTokenFile(keysDir)
//	if err == nil {
//		binding := NewIsyBinding(appConfig, hc)
//		err = binding.Start()
//
//		if err == nil {
//			utils.WaitForSignal(context.Background())
//			binding.Stop()
//			return err
//		}
//	}
//	return err
//}
