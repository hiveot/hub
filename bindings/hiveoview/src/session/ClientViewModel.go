package session

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/services/history/historyclient"
	"sort"
	"time"
)

// ReadDirLimit is the maximum amount of TDs to read in one call
const ReadDirLimit = 1000

// AgentThings holds a collection of Things from an agent
type AgentThings struct {
	AgentID string
	Things  []*things.TD
}

// ClientViewModel for querying and transforming server data for presentation
type ClientViewModel struct {
	hc hubclient.IHubClient
}

// ReadHistory returns historical values of a thing key
func (v *ClientViewModel) ReadHistory(
	thingID string, key string, timestamp time.Time, duration int, limit int) (
	[]*things.ThingMessage, bool, error) {

	hist := historyclient.NewReadHistoryClient(v.hc)
	values, itemsRemaining, err := hist.ReadHistory(
		thingID, key, timestamp, duration, limit)
	return values, itemsRemaining, err
}

// GetLatest returns a map with the latest property values of a thing or nil if failed
// TODO: The generated API doesnt know return types because WoT TD has no
// place to define them. Find a better solution.
func (v *ClientViewModel) GetLatest(thingID string) (things.ThingMessageMap, error) {
	valuesMap := things.NewThingMessageMap()
	tvsJson, err := digitwin.OutboxReadLatest(v.hc, nil, "", "", thingID)
	if err != nil {
		return valuesMap, err
	}
	tvs, _ := things.NewThingMessageMapFromSource(tvsJson)
	for _, tv := range tvs {
		valuesMap.Set(tv.Key, tv)
	}
	return valuesMap, nil
}

// GetTD is a simple helper to retrieve a TD.
// This can re-use a cached version if this model supports caching.
func (v *ClientViewModel) GetTD(thingID string) (*things.TD, error) {
	td := &things.TD{}
	tdJson, err := digitwin.DirectoryReadTD(v.hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJson), &td)
	}
	return td, err
}

// GetValue returns the latest thing message value of an thing event or property
func (v *ClientViewModel) GetValue(thingID string, key string) (*things.ThingMessage, error) {

	// TODO: cache this to avoid multiple reruns
	tmmapJson, err := digitwin.OutboxReadLatest(
		v.hc, []string{key}, vocab.MessageTypeEvent, "", thingID)
	tmmap, _ := things.NewThingMessageMapFromSource(tmmapJson)
	if err != nil {
		return nil, err
	}
	value, found := tmmap[key]
	if !found {
		return nil, fmt.Errorf("key '%s' not found in thing '%s'", key, thingID)
	}
	return value, nil
}

// GroupByAgent groups Things by agent and sorts them by Thing title
func (v *ClientViewModel) GroupByAgent(tds map[string]*things.TD) []*AgentThings {
	agentMap := make(map[string]*AgentThings)
	// first split the things by their agent
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
func (v *ClientViewModel) ReadDirectory() (map[string]*things.TD, error) {
	newThings := make(map[string]*things.TD)

	// TODO: support for paging
	thingsList, err := digitwin.DirectoryReadTDs(v.hc, ReadDirLimit, 0)
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

// SortThingsByTitle as the name suggests sorts the things in the given slice
func (v *ClientViewModel) SortThingsByTitle(tds []*things.TD) {
	sort.Slice(tds, func(i, j int) bool {
		tdI := tds[i]
		tdJ := tds[j]
		return tdI.Title < tdJ.Title
	})
}

func NewClientViewModel(hc hubclient.IHubClient) *ClientViewModel {
	v := &ClientViewModel{hc: hc}
	return v
}
