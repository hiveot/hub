// Package service for basic ISY99x Insteon home automation hub access
// This implements common sensors and switches
package service

import (
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
)

// GetTD generates a TD document for this binding containing properties,
// event and action definitions.
func (svc *IsyBinding) GetTD() *things.TD {
	td := things.NewTD(svc.thingID, "ISY99x binding", vocab.DeviceTypeBinding)

	// TODO: persist configuration
	//binding config
	prop := td.AddProperty(vocab.VocabPollInterval, vocab.VocabPollInterval,
		"Gateway data polling interval", vocab.WoTDataTypeInteger)
	prop.Unit = vocab.UnitNameSecond
	prop.ReadOnly = false
	prop = td.AddProperty(vocab.VocabGatewayAddress, vocab.VocabGatewayAddress,
		"OWServer gateway IP address; empty to auto discover.", vocab.WoTDataTypeString)
	prop.ReadOnly = false

	// gateway events
	//_ = td.AddEvent(vocab.VocabConnected, vocab.VocabConnected, "Connected to OWServer gateway", vocab.WoTDataTypeBool, nil)
	//_ = td.AddEvent(vocab.VocabDisconnected, vocab.VocabDisconnected, "Connected lost to OWServer gateway", vocab.WoTDataTypeBool, nil)

	// no binding actions
	return td
}

//// GetProps returns the property values of the binding Thing
//func (svc *IsyBinding) GetProps(onlyChanges bool) map[string]string {
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
