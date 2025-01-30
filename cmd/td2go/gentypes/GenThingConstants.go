package gentypes

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/td"
)

// GenThingConstantsFromTD generates the thing constants from the TD.
//
// This generates:
//   - thing identifiers:
//     thing agent ID:        {ThingTitle}AgentID = "agentID"
//     thing service ID:      {ThingTitle}ServiceID = "thingID"
//     thing digital twin ID: {ThingTitle}DThingID = "dtw:agentID:thingID"
//   - property names         {ThingTitle}PropName = "name"
//   - event names            {ThingTitle}EventName = "name"
//   - action names           {ThingTitle}ActionName = "name"
func GenThingConstantsFromTD(l *utils.SL, agentID, serviceID string, td1 *td.TD) {
	dThingID := td.MakeDigiTwinThingID(agentID, serviceID)

	// thing identifiers
	l.Add("//--- Constants ---")
	l.Add("")
	l.Add("// %sAgentID is the account ID of the agent managing the Thing.", serviceID)
	l.Add("const %sAgentID = \"%s\"", serviceID, agentID)
	l.Add("")
	l.Add("// %sServiceID is the thingID of the device/service as used by agents.", serviceID)
	l.Add("// Agents use this to publish events and subscribe to actions")
	l.Add("const %sServiceID = \"%s\"", serviceID, serviceID)
	l.Add("")
	l.Add("// %sDThingID is the Digitwin thingID as used by consumers. Digitwin adds the dtw:{agent} prefix to the serviceID", serviceID)
	l.Add("// Consumers use this to publish actions and subscribe to events")
	l.Add("const %sDThingID = \"%s\"", serviceID, dThingID)
	l.Add("")

	// property names
	l.Add("// Property, Event and Action names")
	l.Add("const (")
	l.Indent++
	for name, _ := range td1.Properties {
		nameAsID := Name2ID(name)
		l.Add("%sProp%s = \"%s\"", serviceID, nameAsID, name)
	}
	for name, _ := range td1.Events {
		nameAsID := Name2ID(name)
		l.Add("%sEvent%s = \"%s\"", serviceID, nameAsID, name)
	}
	for name, _ := range td1.Actions {
		nameAsID := Name2ID(name)
		l.Add("%sAction%s = \"%s\"", serviceID, nameAsID, name)
	}
	l.Indent--
	l.Add(")")
}
