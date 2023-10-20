package historyclient

import (
	"github.com/hiveot/hub/core/history"
	"github.com/hiveot/hub/lib/hubclient"
)

// ManageHistoryClient client for managing retention of the history service
type ManageHistoryClient struct {
	// service providing the history capability
	serviceID string
	// capability to use
	capID string
	hc    hubclient.IHubClient
}

// GetRetentionRule returns the retention configuration of an event by name
// This applies to events from any publishers and things
// returns nil if there is no retention rule for the event
//
//	eventName whose retention to return
func (cl *ManageHistoryClient) GetRetentionRule(agentID string, thingID string, name string) (*history.RetentionRule, error) {
	args := history.GetRetentionRuleArgs{
		AgentID: agentID,
		ThingID: thingID,
		Name:    name,
	}
	resp := history.GetRetentionRuleResp{}
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, history.GetRetentionRuleMethod, &args, &resp)
	return resp.Rule, err
}

// GetRetentionRules returns the list of retention rules
func (cl *ManageHistoryClient) GetRetentionRules() (history.RetentionRuleSet, error) {
	resp := history.GetRetentionRulesResp{}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, history.GetRetentionRulesMethod, nil, &resp)
	return resp.Rules, err
}

// SetRetentionRules configures the retention of a Thing event
func (cl *ManageHistoryClient) SetRetentionRules(rules history.RetentionRuleSet) error {
	args := history.SetRetentionRulesArgs{Rules: rules}
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, history.SetRetentionRulesMethod, &args, nil)
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
