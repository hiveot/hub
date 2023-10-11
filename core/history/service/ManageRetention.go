package service

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/pkg/history"
)

// test if ID exists in the array of strings
// returns true if array is empty, eg no values to match
func inArray(arr []string, id string) bool {
	if arr == nil || len(arr) == 0 {
		return true
	}
	for _, s := range arr {
		if s == id {
			return true
		}
	}
	return false
}

// ManageRetention provides the capability to retrieve and edit the event retention configuration.
// This implements the IManageRetention interface.
type ManageRetention struct {
	// defaults from config file
	defaultRetentions []history.EventRetention
	// configuration set through API
	configuredRetentions map[string]history.EventRetention
}

// GetEvents returns the event retention configuration
func (svc *ManageRetention) GetEvents(_ context.Context) ([]history.EventRetention, error) {
	retList := make([]history.EventRetention, 0, len(svc.configuredRetentions))
	for _, ret := range svc.configuredRetentions {
		retList = append(retList, ret)
		logrus.Infof("ret name=%s", ret.Name)
	}
	return retList, nil
}

// GetEventRetention returns the retention configuration of an event by name
// If the event isn't found an error is returned
//
//	eventName whose retention to return
func (svc *ManageRetention) GetEventRetention(
	_ context.Context, eventName string) (ret history.EventRetention, err error) {

	logrus.Infof("")
	evRet, found := svc.configuredRetentions[eventName]
	if !found {
		err = fmt.Errorf("event '%s' not found in the retention list", eventName)
	}
	return evRet, err
}

// Release the capability and its resources
func (svc *ManageRetention) Release() {
	// this is a singleton
}

// RemoveEventRetention removes the retention configuration of an event.
func (svc *ManageRetention) RemoveEventRetention(_ context.Context, eventName string) error {
	logrus.Infof("")
	delete(svc.configuredRetentions, eventName)
	// TODO: save
	return nil
}

// SetEventRetention configures the retention of a Thing event
func (svc *ManageRetention) SetEventRetention(_ context.Context, eventRet history.EventRetention) error {
	logrus.Infof("")
	svc.configuredRetentions[eventRet.Name] = eventRet
	// TODO: save
	return nil
}

// Start the retention manager.
// This loads the retention configuration
func (svc *ManageRetention) Start() error {
	// load default config
	for _, ret := range svc.defaultRetentions {
		svc.configuredRetentions[ret.Name] = ret
	}
	// TODO: load configured retentions from state store

	return nil
}

// Stop using the retention manager
func (svc *ManageRetention) Stop() {
}

// TestEvent tests if the event passes the retention filter rules
// If no rules exist then all events pass.
// returns True if the event passes, false if rejected.
func (svc *ManageRetention) TestEvent(_ context.Context, eventValue thing.ThingValue) (bool, error) {

	rules := svc.configuredRetentions
	// no rules, so accept everything
	hasRetentionRules := rules != nil && len(rules) > 0
	if !hasRetentionRules {
		return true, nil
	}
	// unlisted event are rejected
	rule, found := rules[eventValue.ID]
	if !found {
		return false, nil
	}
	// check publishers, thing IDs
	if !inArray(rule.Publishers, eventValue.PublisherID) ||
		!inArray(rule.Things, eventValue.ThingID) ||
		rule.Exclude != nil && len(rule.Exclude) > 0 && inArray(rule.Exclude, eventValue.ThingID) {
		return false, nil
	}
	return true, nil
}

// NewManageRetention creates a new instance that implements IManageRetention
//
//	defaultConfig with events to retain or nil to use defaults
func NewManageRetention(defaultConfig []history.EventRetention) *ManageRetention {
	if defaultConfig == nil {
		defaultConfig = make([]history.EventRetention, 0)
	}
	svc := &ManageRetention{
		defaultRetentions:    defaultConfig,
		configuredRetentions: make(map[string]history.EventRetention),
	}
	return svc
}
