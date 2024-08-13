package session

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"sort"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
const ReadDirLimit = 1000

// AgentThings holds a collection of Things from an agent
type AgentThings struct {
	AgentID string
	Things  []*things.TD
}

// ClientViewModel generates view model data for rendering user interfaces.
// Some short time minimal caching may take place for performance optimization.
//type ClientViewModel struct {
//	//
//	mux sync.RWMutex
//}

// GroupByAgent groups Things by agent and sorts them by Thing title
func GroupByAgent(tds map[string]*things.TD) map[string]*AgentThings {
	agentMap := make(map[string]*AgentThings)
	for thingID, td := range tds {
		agentID, _ := things.SplitDigiTwinThingID(thingID)
		agentGroup, found := agentMap[agentID]
		if !found {
			agentGroup = &AgentThings{
				AgentID: agentID,
				Things:  make([]*things.TD, 0),
			}
			agentMap[agentID] = agentGroup
		}
		agentGroup.Things = append(agentGroup.Things, td)
	}
	for _, grp := range agentMap {
		SortThingsByTitle(grp.Things)
	}
	return agentMap
}

// ReadDirectory loads and decodes Things from the directory.
// This currently limits the nr of things to ReadDirLimit.
// hc is the connection to use.
func ReadDirectory(hc hubclient.IHubClient) (map[string]*things.TD, error) {
	newThings := make(map[string]*things.TD)

	// TODO: support for paging
	thingsList, err := digitwin.DirectoryReadTDs(hc, ReadDirLimit, 0)
	if err != nil {
		return newThings, err
	}
	for _, tdJson := range thingsList {
		td := things.TD{}
		err = json.Unmarshal([]byte(tdJson), &td)
		if err == nil {
			newThings[td.ID] = &td
		}
	}
	return newThings, nil
}

// ReadTD is a simple helper to read and unmarshal a TD
func ReadTD(hc hubclient.IHubClient, thingID string) (*things.TD, error) {
	td := &things.TD{}
	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJson), &td)
	}
	return td, err
}

// SortThingsByTitle as the name suggests sorts the things in the given slice
func SortThingsByTitle(tds []*things.TD) {
	sort.Slice(tds, func(i, j int) bool {
		tdI := tds[i]
		tdJ := tds[j]
		return tdI.Title < tdJ.Title
	})
}
