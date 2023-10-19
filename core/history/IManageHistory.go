package history

import "github.com/hiveot/hub/lib/thing"

// ManageHistoryCap is the capabilityID for managing history
const ManageHistoryCap = "manageHistory"

// EventNameProperties 'properties' is the name of the event that holds a JSON encoded map
// with one or more property values of a thing.
//const EventNameProperties = vocab.WoTProperties

// RetentionRule with a retention rule for an event (or action)
type RetentionRule struct {
	// Name of the event to record
	Name string `yaml:"name"`

	// Optional, only accept the event from these publishers
	Agents []string `yaml:"agents"`

	// Optional, only accept the event from these things
	Things []string `yaml:"things"`

	// Optional, exclude the event from these things
	Exclude []string `yaml:"exclude"`

	// Retention sets the age of the event in seconds after which it can be removed. 0 for indefinitely (default)
	MaxAge uint64 `yaml:"maxAge"`
}

// CheckRetentionMethod checks if the given event will be retained
const CheckRetentionMethod = "checkRetention"

type CheckRetentionArgs struct {
	// the event value to check
	Event *thing.ThingValue `json:"event"`
}
type CheckRetentionResp struct {
	Retained bool `json:"retained"`
}

// GetRetentionRuleMethod returns the retention configuration of an event by name
// This applies to events from any publishers and things
const GetRetentionRuleMethod = "getRetentionRule"

type GetRetentionRuleArgs struct {
	// Name of the event whose retention settings to get
	Name string `json:"name"`
}
type GetRetentionRuleResp struct {
	// The event retention if successful
	Rule *RetentionRule `json:"eventRetention"`
}

// GetRetentionRulesMethod returns the collection of retention configurations
const GetRetentionRulesMethod = "getRetentionRules"

type GetRetentionRulesResp struct {
	Rules []*RetentionRule `json:"rules"`
}

// RemoveRetentionRuleMethod removes an existing event retention rule
// If the rule doesn't exist this is considered successful and no error will be returned
const RemoveRetentionRuleMethod = "removeRetentionRule"

type RemoveRetentionRuleArgs struct {
	// Name of the event whose retention settings to remove
	Name string `json:"name"`
}

// SetRetentionRuleMethod configures the retention of a Thing event
const SetRetentionRuleMethod = "setRetentionRule"

type SetRetentionRuleArgs struct {
	Rule *RetentionRule `json:"rule"`
}
