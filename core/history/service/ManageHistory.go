package service

import (
	"fmt"
	"github.com/hiveot/hub/core/history"
	"github.com/hiveot/hub/lib/hubclient"
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

// ManageHistory provides the capability to manage how history is captured
type ManageHistory struct {
	// default retention rules
	defaultRetentions []*history.RetentionRule
	// configuration set through API
	configuredRetentions map[string]*history.RetentionRule
	//
	retSub hubclient.ISubscription
	//
	hc hubclient.IHubClient
}

// CheckRetention tests if the event passes the retention filter rules
// If no rules exist then all events pass.
// returns True if the event passes, false if rejected.
func (svc *ManageHistory) CheckRetention(
	clientID string, tv *thing.ThingValue) (bool, error) {
	//resp := &history.CheckRetentionResp{Retained: false}

	rules := svc.configuredRetentions
	// no rules, so accept everything
	hasRetentionRules := rules != nil && len(rules) > 0
	if !hasRetentionRules {
		return true, nil
	}
	// unlisted event are rejected
	rule, found := rules[tv.Name]
	if !found {
		return false, nil
	}

	// check publishers, thing IDs
	if !inArray(rule.Agents, tv.AgentID) ||
		!inArray(rule.Things, tv.ThingID) ||
		rule.Exclude != nil && len(rule.Exclude) > 0 && inArray(rule.Exclude, tv.ThingID) {
		return false, nil
	}

	return true, nil
}

// GetRetentionRule returns the retention rule of an event by name
// If the event isn't found an error is returned
//
//	eventName whose retention to return
func (svc *ManageHistory) GetRetentionRule(
	clientID string, name string) (resp *history.RetentionRule, err error) {

	rule, found := svc.configuredRetentions[name]
	if !found {
		err = fmt.Errorf("rule '%s' not found in the retention list", name)
	}
	//resp = &history.GetRetentionRuleResp{Rule: evRet}
	return rule, err
}

// GetRetentionRules returns all retention rules
func (svc *ManageHistory) GetRetentionRules() ([]*history.RetentionRule, error) {
	retList := make([]*history.RetentionRule, 0, len(svc.configuredRetentions))
	for _, ret := range svc.configuredRetentions {
		retList = append(retList, ret)
		slog.Info("GetEvents", slog.String("name", ret.Name))
	}
	//resp = &history.GetRetentionRulesResp{Rules: retList}
	return retList, nil
}

// RemoveRetentionRule removes the retention rule for an event.
func (svc *ManageHistory) RemoveRetentionRule(clientID string, name string) error {

	slog.Info("RemoveEventRetention", "clientID", clientID, "name", name)
	delete(svc.configuredRetentions, name)
	// TODO: save
	return nil
}

// SetRetentionRule configures the retention of a Thing event
func (svc *ManageHistory) SetRetentionRule(clientID string, rule *history.RetentionRule) error {

	slog.Info("SetEventRetention")
	svc.configuredRetentions[rule.Name] = rule
	// TODO: save
	return nil
}

// Start the history management handler.
// This loads the retention configuration
func (svc *ManageHistory) Start() (err error) {
	// load default config
	for _, ret := range svc.defaultRetentions {
		svc.configuredRetentions[ret.Name] = ret
	}
	// TODO: load configured retentions from state store
	capMethods := map[string]interface{}{
		history.CheckRetentionMethod:      svc.CheckRetention,
		history.GetRetentionRuleMethod:    svc.GetRetentionRule,
		history.GetRetentionRulesMethod:   svc.GetRetentionRules,
		history.RemoveRetentionRuleMethod: svc.RemoveRetentionRule,
		history.SetRetentionRuleMethod:    svc.SetRetentionRule,
	}
	svc.retSub, err = hubclient.SubRPCCapability(
		svc.hc, history.ManageHistoryCap, capMethods)
	return nil
}

// Stop using the retention manager
func (svc *ManageHistory) Stop() {
	svc.retSub.Unsubscribe()
}

// NewManageRetention creates a new instance that implements IManageRetention
//
//	defaultConfig with events to retain or nil to use defaults
func NewManageRetention(hc hubclient.IHubClient, defaultConfig []*history.RetentionRule) *ManageHistory {
	if defaultConfig == nil {
		defaultConfig = make([]*history.RetentionRule, 0)
	}
	svc := &ManageHistory{
		hc:                   hc,
		defaultRetentions:    defaultConfig,
		configuredRetentions: make(map[string]*history.RetentionRule),
	}
	return svc
}
