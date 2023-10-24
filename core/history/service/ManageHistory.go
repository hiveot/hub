package service

import (
	"github.com/hiveot/hub/core/history/historyapi"
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
	// retention rules grouped by event ID
	rules historyapi.RetentionRuleSet
	//
	retSub hubclient.ISubscription
	//
	hc hubclient.IHubClient
}

// return the first retention rule that applies to the given value or nil if no rule applies
func (svc *ManageHistory) _FindFirstRule(tv *thing.ThingValue) *historyapi.RetentionRule {
	// two sets of rules apply, those that match the name and those that don't filter by name
	// rules with specified event names take precedence
	rules1, found := svc.rules[tv.Name]
	if found {
		// there is a potential to optimize this for a lot of rules by
		// include a nested map of agentIDs and ThingIDs for fast lookup.
		// before going down that road some performance analysis needs to be done first
		for _, rule := range rules1 {
			if (rule.AgentID == "" || rule.AgentID == tv.AgentID) &&
				(rule.ThingID == "" || rule.ThingID == tv.ThingID) {
				return rule
			}
		}
	}
	// rules that apply to any event/action names
	rules2, found := svc.rules[""]
	if found {
		for _, rule := range rules2 {
			if (rule.AgentID == "" || rule.AgentID == tv.AgentID) &&
				(rule.ThingID == "" || rule.ThingID == tv.ThingID) {
				return rule
			}
		}
	}
	// no applicable rule found
	return nil
}

// _IsRetained returns the rule 'Retain' flag if a matching rule is found
// If no retention rules are defined this returns true
// If rules are defined but not found this returns false
func (svc *ManageHistory) _IsRetained(tv *thing.ThingValue) (bool, *historyapi.RetentionRule) {
	if svc.rules == nil || len(svc.rules) == 0 {
		return true, nil
	}
	rule := svc._FindFirstRule(tv)
	if rule == nil {
		return false, nil
	}
	return rule.Retain, rule
}

// GetRetentionRule returns the first retention rule that applies
// to the given value.
// This returns nil without error if no retention rules are defined.
//
//	eventName whose retention to return
func (svc *ManageHistory) GetRetentionRule(
	ctx hubclient.ServiceContext, args *historyapi.GetRetentionRuleArgs) (resp *historyapi.GetRetentionRuleResp, err error) {

	tv := thing.ThingValue{
		AgentID: args.AgentID,
		ThingID: args.ThingID,
		Name:    args.Name,
	}
	rule := svc._FindFirstRule(&tv)
	resp = &historyapi.GetRetentionRuleResp{Rule: rule}
	return resp, err
}

// GetRetentionRules returns all retention rules
func (svc *ManageHistory) GetRetentionRules() (*historyapi.GetRetentionRulesResp, error) {
	resp := &historyapi.GetRetentionRulesResp{Rules: svc.rules}
	return resp, nil
}

// SetRetentionRules updates the retention rules set
func (svc *ManageHistory) SetRetentionRules(
	ctx hubclient.ServiceContext, args *historyapi.SetRetentionRulesArgs) error {
	ruleCount := 0
	// ensure that the name in the rule matches the key in the map
	for name, nameRules := range args.Rules {
		for _, rule := range nameRules {
			rule.Name = name
			ruleCount++
		}
	}

	slog.Info("SetRetentionRules", slog.Int("nr-rules", ruleCount))
	svc.rules = args.Rules
	return nil
}

// Start the history management handler.
// This loads the retention configuration
func (svc *ManageHistory) Start() (err error) {

	// TODO: load latest retention rules from state store
	capMethods := map[string]interface{}{
		historyapi.GetRetentionRuleMethod:  svc.GetRetentionRule,
		historyapi.GetRetentionRulesMethod: svc.GetRetentionRules,
		historyapi.SetRetentionRulesMethod: svc.SetRetentionRules,
	}
	svc.retSub, err = hubclient.SubRPCCapability(
		svc.hc, historyapi.ManageHistoryCap, capMethods)
	return nil
}

// Stop using the retention manager
func (svc *ManageHistory) Stop() {
	svc.retSub.Unsubscribe()
}

// NewManageRetention creates a new instance that implements IManageRetention
//
//	defaultRules with rules from config
func NewManageRetention(
	hc hubclient.IHubClient, defaultRules historyapi.RetentionRuleSet) *ManageHistory {
	if defaultRules == nil {
		defaultRules = make(historyapi.RetentionRuleSet)
	}
	svc := &ManageHistory{
		hc:    hc,
		rules: defaultRules,
	}
	return svc
}
