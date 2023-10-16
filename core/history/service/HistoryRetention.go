package service

import (
	"context"
	"fmt"
	"github.com/hiveot/hub/core/history"
	"github.com/hiveot/hub/lib/thing"
	"log/slog"
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

// HistoryRetention provides the capability to retrieve and edit the event retention configuration.
// This implements the IManageRetention interface.
type HistoryRetention struct {
	// defaults from config file
	defaultRetentions []history.RetentionRule
	// configuration set through API
	configuredRetentions map[string]history.RetentionRule
}

// CheckRetention tests if the event passes the retention filter rules
// If no rules exist then all events pass.
// returns True if the event passes, false if rejected.
func (svc *HistoryRetention) CheckRetention(evMsg *thing.ThingValue) (bool, error) {

	rules := svc.configuredRetentions
	// no rules, so accept everything
	hasRetentionRules := rules != nil && len(rules) > 0
	if !hasRetentionRules {
		return true, nil
	}
	// unlisted event are rejected
	rule, found := rules[evMsg.Name]
	if !found {
		return false, nil
	}
	// check publishers, thing IDs
	if !inArray(rule.Publishers, evMsg.AgentID) ||
		!inArray(rule.Things, evMsg.ThingID) ||
		rule.Exclude != nil && len(rule.Exclude) > 0 && inArray(rule.Exclude, evMsg.ThingID) {
		return false, nil
	}
	return true, nil
}

// GetEvents returns the event retention configuration
func (svc *HistoryRetention) GetEvents() ([]history.RetentionRule, error) {
	retList := make([]history.RetentionRule, 0, len(svc.configuredRetentions))
	for _, ret := range svc.configuredRetentions {
		retList = append(retList, ret)
		slog.Info("GetEvents", slog.String("name", ret.Name))
	}
	return retList, nil
}

// GetEventRetention returns the retention configuration of an event by name
// If the event isn't found an error is returned
//
//	eventName whose retention to return
func (svc *HistoryRetention) GetEventRetention(
	_ context.Context, eventName string) (ret history.RetentionRule, err error) {

	slog.Info("GetEventRetention")
	evRet, found := svc.configuredRetentions[eventName]
	if !found {
		err = fmt.Errorf("event '%s' not found in the retention list", eventName)
	}
	return evRet, err
}

// Release the capability and its resources
func (svc *HistoryRetention) Release() {
	// this is a singleton
}

// RemoveEventRetention removes the retention configuration of an event.
func (svc *HistoryRetention) RemoveEventRetention(eventName string) error {
	slog.Info("RemoveEventRetention")
	delete(svc.configuredRetentions, eventName)
	// TODO: save
	return nil
}

// SetEventRetention configures the retention of a Thing event
func (svc *HistoryRetention) SetEventRetention(eventRet history.RetentionRule) error {
	slog.Info("SetEventRetention")
	svc.configuredRetentions[eventRet.Name] = eventRet
	// TODO: save
	return nil
}

// Start the retention manager.
// This loads the retention configuration
func (svc *HistoryRetention) Start() error {
	// load default config
	for _, ret := range svc.defaultRetentions {
		svc.configuredRetentions[ret.Name] = ret
	}
	// TODO: load configured retentions from state store

	return nil
}

// Stop using the retention manager
func (svc *HistoryRetention) Stop() {
}

// NewManageRetention creates a new instance that implements IManageRetention
//
//	defaultConfig with events to retain or nil to use defaults
func NewManageRetention(defaultConfig []history.RetentionRule) *HistoryRetention {
	if defaultConfig == nil {
		defaultConfig = make([]history.RetentionRule, 0)
	}
	svc := &HistoryRetention{
		defaultRetentions:    defaultConfig,
		configuredRetentions: make(map[string]history.RetentionRule),
	}
	return svc
}
