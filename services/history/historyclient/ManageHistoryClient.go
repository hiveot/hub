package historyclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/services/history/historyapi"
	"github.com/hiveot/hub/wot/tdd"
)

// ManageHistoryClient client for managing retention of the history service
type ManageHistoryClient struct {
	// service providing the history management capability
	dThingID string
	hc       hubclient.IConsumerClient
}

// GetRetentionRule returns the retention configuration of an event by name
// This applies to events from any publishers and things
// returns nil if there is no retention rule for the event
//
//	dThingID
//	eventName whose retention to return
func (cl *ManageHistoryClient) GetRetentionRule(dThingID string, name string) (*historyapi.RetentionRule, error) {
	args := historyapi.GetRetentionRuleArgs{
		ThingID: dThingID,
		Name:    name,
	}
	resp := historyapi.GetRetentionRuleResp{}
	err := cl.hc.Rpc(cl.dThingID, historyapi.GetRetentionRuleMethod, &args, &resp)
	return resp.Rule, err
}

// GetRetentionRules returns the list of retention rules
func (cl *ManageHistoryClient) GetRetentionRules() (historyapi.RetentionRuleSet, error) {
	resp := historyapi.GetRetentionRulesResp{}
	err := cl.hc.Rpc(cl.dThingID, historyapi.GetRetentionRulesMethod, nil, &resp)
	return resp.Rules, err
}

// SetRetentionRules configures the retention of a Thing event
func (cl *ManageHistoryClient) SetRetentionRules(rules historyapi.RetentionRuleSet) error {
	args := historyapi.SetRetentionRulesArgs{Rules: rules}
	err := cl.hc.Rpc(cl.dThingID, historyapi.SetRetentionRulesMethod, &args, nil)
	return err
}

// NewManageHistoryClient creates a new instance of the manage history client for use by authorized clients
func NewManageHistoryClient(hc hubclient.IConsumerClient) *ManageHistoryClient {
	agentID := historyapi.AgentID
	mngCl := &ManageHistoryClient{
		dThingID: tdd.MakeDigiTwinThingID(agentID, historyapi.ManageHistoryServiceID),
		hc:       hc,
	}
	return mngCl
}
