package historyclient

import (
	"github.com/hiveot/hub/core/history/historyapi"
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
func (cl *ManageHistoryClient) GetRetentionRule(agentID string, thingID string, name string) (*historyapi.RetentionRule, error) {
	args := historyapi.GetRetentionRuleArgs{
		AgentID: agentID,
		ThingID: thingID,
		Name:    name,
	}
	resp := historyapi.GetRetentionRuleResp{}
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, historyapi.GetRetentionRuleMethod, &args, &resp)
	return resp.Rule, err
}

// GetRetentionRules returns the list of retention rules
func (cl *ManageHistoryClient) GetRetentionRules() (historyapi.RetentionRuleSet, error) {
	resp := historyapi.GetRetentionRulesResp{}
	_, err := cl.hc.PubRPCRequest(cl.serviceID, cl.capID, historyapi.GetRetentionRulesMethod, nil, &resp)
	return resp.Rules, err
}

// SetRetentionRules configures the retention of a Thing event
func (cl *ManageHistoryClient) SetRetentionRules(rules historyapi.RetentionRuleSet) error {
	args := historyapi.SetRetentionRulesArgs{Rules: rules}
	_, err := cl.hc.PubRPCRequest(
		cl.serviceID, cl.capID, historyapi.SetRetentionRulesMethod, &args, nil)
	return err
}

// NewManageHistoryClient creates a new instance of the manage history client for use by authorized clients
func NewManageHistoryClient(hc hubclient.IHubClient) *ManageHistoryClient {
	mngCl := &ManageHistoryClient{
		serviceID: historyapi.ServiceName,
		capID:     historyapi.ManageHistoryCap,
		hc:        hc,
	}
	return mngCl
}
