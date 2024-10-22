package tdd2go

import (
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
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
func GenThingConstantsFromTD(l *utils.SL, thingTitleID string, td *tdd.TD) {
	dThingID := td.ID
	agentID, serviceID := tdd.SplitDigiTwinThingID(dThingID)

	// thing identifiers
	l.Add("// %sAgentID is the account ID of the agent managing the Thing.", thingTitleID)
	l.Add("const %sAgentID = \"%s\"", thingTitleID, agentID)
	l.Add("// %sServiceID is the thingID of the device/service as used by agents.", thingTitleID)
	l.Add("// Agents use this to publish events and subscribe to actions")
	l.Add("const %sServiceID = \"%s\"", thingTitleID, serviceID)
	l.Add("// %sDThingID is the Digitwin thingID as used by consumers. Digitwin adds the dtw:{agent} prefix to the serviceID", thingTitleID)
	l.Add("// Consumers use this to publish actions and subscribe to events")
	l.Add("const %sDThingID = \"%s\"", thingTitleID, dThingID)
	l.Add("")

	// property names
	l.Add("// Thing names")
	l.Add("const (")
	l.Indent++
	for name, _ := range td.Properties {
		nameAsID := Name2ID(name)
		l.Add("%sProp%s = \"%s\"", thingTitleID, nameAsID, name)
	}
	for name, _ := range td.Events {
		nameAsID := Name2ID(name)
		l.Add("%sEvent%s = \"%s\"", thingTitleID, nameAsID, name)
	}
	for name, _ := range td.Actions {
		nameAsID := Name2ID(name)
		l.Add("%sAction%s = \"%s\"", thingTitleID, nameAsID, name)
	}
	l.Indent--
	l.Add(")")
}
