package session

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
	"sort"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
//const ReadDirLimit = 1000

// AgentThings holds a collection of Things from an agent
type AgentThings struct {
	AgentID string
	Things  []*tdd.TD
}

// ClientViewModel for querying and transforming server data for presentation
type ClientViewModel struct {
	// connection with the hub
	hc hubclient.IConsumerClient
}

// GroupByAgent groups Things by agent and sorts them by Thing title
func (v *ClientViewModel) GroupByAgent(tds map[string]*tdd.TD) []*AgentThings {
	agentMap := make(map[string]*AgentThings)
	// first split the things by their agent
	for thingID, td := range tds {
		agentID, _ := tdd.SplitDigiTwinThingID(thingID)
		agentGroup, found := agentMap[agentID]
		if !found {
			agentGroup = &AgentThings{
				AgentID: agentID,
				Things:  make([]*tdd.TD, 0),
			}
			agentMap[agentID] = agentGroup
		}
		agentGroup.Things = append(agentGroup.Things, td)
	}
	// next, sort the agent things
	agentsList := make([]*AgentThings, 0, len(agentMap))
	for _, grp := range agentMap {
		agentsList = append(agentsList, grp)
		v.SortThingsByTitle(grp.Things)
	}
	// last sort the agents
	sort.Slice(agentsList, func(i, j int) bool {
		return agentsList[i].AgentID < agentsList[j].AgentID
	})
	return agentsList
}

// ReadDirectory loads and decodes Things from the directory.
// This currently limits the nr of things to ReadDirLimit.
//func (v *ClientViewModel) ReadDirectory() (map[string]*tdd.TD, error) {
//	newThings := make(map[string]*tdd.TD)
//
//	// TODO: support for paging
//	thingsList, err := digitwin.DirectoryReadTDs(v.hc, ReadDirLimit, 0)
//	if err != nil {
//		return newThings, err
//	}
//	for _, tdJson := range thingsList {
//		td := tdd.TD{}
//		err = json.Unmarshal([]byte(tdJson), &td)
//		if err == nil {
//			newThings[td.ID] = &td
//		}
//	}
//	return newThings, nil
//}

// SortThingsByTitle as the name suggests sorts the things in the given slice
func (v *ClientViewModel) SortThingsByTitle(tds []*tdd.TD) {
	sort.Slice(tds, func(i, j int) bool {
		tdI := tds[i]
		tdJ := tds[j]
		return tdI.Title < tdJ.Title
	})
}

func NewClientViewModel(hc hubclient.IConsumerClient) *ClientViewModel {
	v := &ClientViewModel{hc: hc}
	return v
}
