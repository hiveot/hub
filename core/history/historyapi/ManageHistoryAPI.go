package historyapi

// ManageHistoryCap is the capabilityID for managing history
const ManageHistoryCap = "manageHistory"

// RetentionRule with a retention rule for an event (or action)
type RetentionRule struct {
	// Optional, the rule applies to data from this agent
	AgentID string `yaml:"agentID,omitempty" json:"agentID,omitempty"`

	// Optional, the rule applies to data from this things
	ThingID string `yaml:"thingID,omitempty" json:"thingID,omitempty"`

	// Optional, the rule applies to events or actions with this name
	Name string `yaml:"name"`

	// TODO: class of value, eg @type in the TD (eg, temperature, humidity)
	//Type string `yaml:"type",json:"type"`

	// Retain or not retain based on this rule
	Retain bool `yaml:"retain" json:"retain"`

	// Retention age after which to remove the value. 0 to retain indefinitely
	MaxAge uint64 `yaml:"maxAge"`
}

// RetentionRuleSet is a map by event/action name with one or more rules for agent/things.
type RetentionRuleSet map[string][]*RetentionRule

// GetRetentionRuleMethod returns the first retention rule that applies
// to the given value.
const GetRetentionRuleMethod = "getRetentionRule"

type GetRetentionRuleArgs struct {
	// AgentID is optional
	AgentID string `json:"agentID,omitempty"`
	// ThingID is optional
	ThingID string `json:"thingID,omitempty"`
	// Name of the event whose retention settings to get
	Name string `json:"name,omitempty"`
}
type GetRetentionRuleResp struct {
	Rule *RetentionRule `json:"rule"`
}

// GetRetentionRulesMethod returns the collection of retention configurations
const GetRetentionRulesMethod = "getRetentionRules"

type GetRetentionRulesResp struct {
	Rules RetentionRuleSet `json:"rules"`
}

// SetRetentionRulesMethod updates the set of retention rules
const SetRetentionRulesMethod = "setRetentionRules"

type SetRetentionRulesArgs struct {
	Rules RetentionRuleSet `json:"rules"`
}
