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
	args := history.CheckRetentionArgs{Event: eventValue}
	resp := history.CheckRetentionResp{}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, history.CheckRetentionMethod, &args, &resp)
	return resp.Retained, err
}

// GetRetentionRule returns the retention configuration of an event by name
// This applies to events from any publishers and things
// returns nil if there is no retention rule for the event
//
//	eventName whose retention to return
func (cl *ManageHistoryClient) GetRetentionRule(name string) (*history.RetentionRule, error) {
	args := history.GetRetentionRuleArgs{Name: name}
	resp := history.GetRetentionRuleResp{}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, history.GetRetentionRuleMethod, &args, &resp)
	return resp.Rule, err
}

// GetRetentionRules returns the list of retention rules
func (cl *ManageHistoryClient) GetRetentionRules() ([]*history.RetentionRule, error) {
	resp := history.GetRetentionRulesResp{}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, history.GetRetentionRulesMethod, nil, &resp)
	return resp.Rules, err
}

// RemoveRetentionRule removes an existing event retention rule
// If the rule doesn't exist this is considered successful and no error will be returned
func (cl *ManageHistoryClient) RemoveRetentionRule(name string) error {
	args := history.RemoveRetentionRuleArgs{Name: name}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, history.RemoveRetentionRuleMethod, &args, nil)
	return err
}

// SetRetentionRule configures the retention of a Thing event
func (cl *ManageHistoryClient) SetRetentionRule(eventRetention *history.RetentionRule) error {
	args := history.SetRetentionRuleArgs{Rule: eventRetention}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, history.SetRetentionRuleMethod, &args, nil)
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
