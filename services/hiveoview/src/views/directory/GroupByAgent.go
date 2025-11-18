package directory

import (
	"sort"

	"github.com/hiveot/hivekitgo/wot/td"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
//const ReadDirLimit = 1000

// AgentThings holds a collection of Things from an agent
type AgentThings struct {
	AgentID string
	Things  []*td.TD
}

// GroupByAgent groups Things by agent and sorts them by Thing title
func GroupByAgent(tds map[string]*td.TD) []*AgentThings {
	agentMap := make(map[string]*AgentThings)
	// first split the things by their agent
	for thingID, tdi := range tds {
		agentID, _ := td.SplitDigiTwinThingID(thingID)
		agentGroup, found := agentMap[agentID]
		if !found {
			agentGroup = &AgentThings{
				AgentID: agentID,
				Things:  make([]*td.TD, 0),
			}
			agentMap[agentID] = agentGroup
		}
		agentGroup.Things = append(agentGroup.Things, tdi)
	}
	// next, sort the agent things
	agentsList := make([]*AgentThings, 0, len(agentMap))
	for _, grp := range agentMap {
		agentsList = append(agentsList, grp)
		td.SortThingsByTitle(grp.Things)
	}
	// last sort the agents
	sort.Slice(agentsList, func(i, j int) bool {
		return agentsList[i].AgentID < agentsList[j].AgentID
	})
	return agentsList
}
