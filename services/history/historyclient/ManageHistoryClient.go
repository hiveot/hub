package historyclient

import (
	"github.com/hiveot/hub/services/history/historyapi"
	"github.com/hiveot/hub/transports/consumer"
	"github.com/hiveot/hub/wot/td"
)

// ManageHistoryClient client for managing retention of the history service
type ManageHistoryClient struct {
	// service providing the history management capability
	dThingID string
	//co       transports.IClientConnection
	co *consumer.Consumer
}

// GetRetentionRule returns the retention configuration of an event by name
// This applies to events from any publishers and things
// returns nil if there is no retention rule for the event
//
//	dThingID
//	eventName whose retention to return
func (cl *ManageHistoryClient) GetRetentionRule(
	dThingID string, name string) (*historyapi.RetentionRule, error) {

	args := historyapi.GetRetentionRuleArgs{
		ThingID: dThingID,
		Name:    name,
	}
	resp := historyapi.GetRetentionRuleResp{}
	err := cl.co.InvokeAction(cl.dThingID, historyapi.GetRetentionRuleMethod, &args, &resp)
	return resp.Rule, err
}

// GetRetentionRules returns the list of retention rules
func (cl *ManageHistoryClient) GetRetentionRules() (historyapi.RetentionRuleSet, error) {
	resp := historyapi.GetRetentionRulesResp{}
	err := cl.co.InvokeAction(cl.dThingID, historyapi.GetRetentionRulesMethod, nil, &resp)
	return resp.Rules, err
}

// SetRetentionRules configures the retention of a Thing event
func (cl *ManageHistoryClient) SetRetentionRules(rules historyapi.RetentionRuleSet) error {
	args := historyapi.SetRetentionRulesArgs{Rules: rules}
	err := cl.co.InvokeAction(cl.dThingID, historyapi.SetRetentionRulesMethod, &args, nil)
	return err
}

// NewManageHistoryClient creates a new instance of the manage history client for use by authorized clients
func NewManageHistoryClient(co *consumer.Consumer) *ManageHistoryClient {
	agentID := historyapi.AgentID
	mngCl := &ManageHistoryClient{
		dThingID: td.MakeDigiTwinThingID(agentID, historyapi.ManageHistoryServiceID),
		co:       co,
	}
	return mngCl
}
