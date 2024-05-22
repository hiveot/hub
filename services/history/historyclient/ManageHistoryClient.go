package historyclient

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/history/historyapi"
)

// ManageHistoryClient client for managing retention of the history service
type ManageHistoryClient struct {
	// service providing the history management capability
	dThingID string
	hc       hubclient.IHubClient
}

// GetRetentionRule returns the retention configuration of an event by name
// This applies to events from any publishers and things
// returns nil if there is no retention rule for the event
//
//	eventName whose retention to return
func (cl *ManageHistoryClient) GetRetentionRule(dThingID string, key string) (*historyapi.RetentionRule, error) {
	args := historyapi.GetRetentionRuleArgs{
		ThingID: dThingID,
		Key:     key,
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
func NewManageHistoryClient(hc hubclient.IHubClient) *ManageHistoryClient {
	agentID := historyapi.AgentID
	mngCl := &ManageHistoryClient{
		dThingID: things.MakeDigiTwinThingID(agentID, historyapi.ManageHistoryServiceID),
		hc:       hc,
	}
	return mngCl
}
