package historyapi

// AgentID is the agentID of the history services
const AgentID = "history"

// ManageHistoryServiceID is the ID of the service exposed by the agent
const ManageHistoryServiceID = "manage"

// ManageHistoryThingID is the ThingID of the manage history service
//var ManageHistoryThingID = things.MakeDigiTwinThingID(HistoryAgentID, ManageHistoryCap)

// Management methods
const (
	// GetRetentionRuleMethod returns the first retention rule that applies
	// to the given value.
	GetRetentionRuleMethod = "getRetentionRule"

	// GetRetentionRulesMethod returns the collection of retention configurations
	GetRetentionRulesMethod = "getRetentionRules"

	// SetRetentionRulesMethod updates the set of retention rules
	SetRetentionRulesMethod = "setRetentionRules"
)

// RetentionRule with a retention rule for an event (or action)
type RetentionRule struct {
	// Optional, the rule applies to data from this (digital twin) Thing
	ThingID string `yaml:"thingID,omitempty" json:"thingID,omitempty"`

	// Optional, the rule applies to events or actions with this name
	Name string `yaml:"name,omitempty"`

	// TODO: class of value, eg @type in the TD (eg, temperature, humidity)
	//Type string `yaml:"type",json:"type"`

	// Retain or not retain based on this rule
	Retain bool `yaml:"retain" json:"retain"`

	// Retention age after which to remove the value. 0 to retain indefinitely
	MaxAge uint64 `yaml:"maxAge"`
}

// RetentionRuleSet is a map by event/action name with one or more rules for agent/things.
type RetentionRuleSet map[string][]*RetentionRule

type GetRetentionRuleArgs struct {
	// ThingID whose rule to get (digital twin ID) is optional
	ThingID string `json:"thingID,omitempty"`
	// Name of the event whose retention settings to get
	Name string `json:"name,omitempty"`
	// Retention for events,properties,actions or empty for all
	AffordanceType string `json:"affordanceType,omitempty"`
}
type GetRetentionRuleResp struct {
	Rule *RetentionRule `json:"rule"`
}

type GetRetentionRulesResp struct {
	Rules RetentionRuleSet `json:"rules"`
}

type SetRetentionRulesArgs struct {
	Rules RetentionRuleSet `json:"rules"`
}
