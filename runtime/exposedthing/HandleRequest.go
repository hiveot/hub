package exposedthing

import (
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/wot/td"
)

/*
Helper function to generate a TD for agents and handle incoming requests.
Objective:
1. One place to define the TD for use in json and code
2. Easy to add action/property write handlers
3. Easy to publish event/prop updates


Option 1: define TD in code with go-generate

Input: Server interface with annotations
Output: TD json file
Output: Client source code in golang
Output: Client source code in other languages

Example:
// @title: Action Status
// @description: Status of the last action
type ActionStatus struct {
	@title: Agent ID
	@description: The agent handling the action
	AgentID string `json:"agentID"`

	@enum: "completed", "failed", "running", "pending"
	Status string `json:"status"`

	@title Creation Time
	@type: dateTime          // type override for TD
	TimeRequested string `json:"timeRequested"`
}

// Action API
// @title Thing title
// @description...
// @type context type
type IService interface {
  ReadProperty(thingID string, name string) (value any, error)
}

*/

type ThingAction struct {
	Name    string
	Handler func(message *messaging.RequestMessage) *messaging.ResponseMessage
	Input   *td.DataSchema // reuse common schemas
	Output  *td.DataSchema
}

/* Use go-generate on a td json file?
pro: similar to tdd2go but per package;
pro: easy to use
con: complex
*/

/* Using go-generate for properties:
Define  Property type
type Property1 struct {
	name string
	title string
	safe bool
	readOnly bool
}
(p *Property1) GetAffordance() string {Marshal(p)}
Generate: property affordance
*/

/* Using go-generate for actions:
Define Service:  Method1(arg1, arg2, arg3...): (res1,error)
Generate: Identical client API
Generate: to/from Request/Response message
Generate: action affordance

*/
