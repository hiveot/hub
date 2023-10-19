package historyclient

import (
	"github.com/hiveot/hub/core/history"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
)

// ManageHistoryClient client for managing retention of the history service
type ManageHistoryClient struct {
	// service providing the history capability
	serviceID string
	// capability to use
	capID string
	hc    hubclient.IHubClient
}

// CheckRetention tests if the event will be retained
func (cl *ManageHistoryClient) CheckRetention(eventValue *thing.ThingValue) (bool, error) {
	//args := history.CheckRetentionArgs{Event: eventValue}
	//resp := history.CheckRetentionResp{}
	resp := false
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, history.CheckRetentionMethod, eventValue, &resp)
	return resp, err
}

// GetRetentionRule returns the retention configuration of an event by name
// This applies to events from any publishers and things
// returns nil if there is no retention rule for the event
//
//	eventName whose retention to return
func (cl *ManageHistoryClient) GetRetentionRule(name string) (*history.RetentionRule, error) {
	//args := history.GetRetentionRuleArgs{Name: name}
	//resp := history.GetRetentionRuleResp{}
	resp := &history.RetentionRule{}
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, history.GetRetentionRuleMethod, name, &resp)
	return resp, err
}

// GetRetentionRules returns the list of retention rules
func (cl *ManageHistoryClient) GetRetentionRules() ([]*history.RetentionRule, error) {
	//resp := history.GetRetentionRulesResp{}
	rules := make([]*history.RetentionRule, 0)
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, history.GetRetentionRulesMethod, nil, &rules)
	return rules, err
}

// RemoveRetentionRule removes an existing event retention rule
// If the rule doesn't exist this is considered successful and no error will be returned
func (cl *ManageHistoryClient) RemoveRetentionRule(name string) error {
	//args := history.RemoveRetentionRuleArgs{Name: name}
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, history.RemoveRetentionRuleMethod, name, nil)
	return err
}

// SetRetentionRule configures the retention of a Thing event
func (cl *ManageHistoryClient) SetRetentionRule(rule *history.RetentionRule) error {
	//args := history.SetRetentionRuleArgs{Rule: eventRetention}
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, history.SetRetentionRuleMethod, rule, nil)
	return err
}

// NewManageHistoryClient creates a new instance of the manage history client for use by authorized clients
func NewManageHistoryClient(hc hubclient.IHubClient) *ManageHistoryClient {
	mngCl := &ManageHistoryClient{
		serviceID: history.ServiceName,
		capID:     history.ManageHistoryCap,
		hc:        hc,
	}
	return mngCl
}
