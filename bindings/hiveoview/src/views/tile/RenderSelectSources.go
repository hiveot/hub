package tile

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"net/http"
)

const RenderSelectSourceTemplateFile = "RenderSelectSources.gohtml"

type RenderSelectSourcesTemplateData struct {
	AgentThings []*session.AgentThings
	// map of thing latest event values
	Values map[string]hubclient.ThingMessageMap
}

// GetUpdated returns the update timestamp of the latest event value
func (data RenderSelectSourcesTemplateData) GetUpdated(thingID string, key string) string {
	tv, found := data.Values[thingID]
	if !found {
		return ""
	}
	tm, found := tv[key]
	if !found {
		return ""
	}
	return tm.GetUpdated()
}

// GetValue returns the string value of a thing event
func (data RenderSelectSourcesTemplateData) GetValue(thingID string, key string) string {
	tv, found := data.Values[thingID]
	if !found {
		return ""
	}
	tm, found := tv[key]
	if !found {
		return ""
	}
	return tm.DataAsText()
}

// RenderSelectSources renders the selection of Tile sources for adding to a tile
func RenderSelectSources(w http.ResponseWriter, r *http.Request) {

	sess, _, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	vm := sess.GetViewModel()
	cts := sess.GetConsumedThingsSession()
	// provide a list of things for each agent and show a nested
	// list of events per thing: agent -> thing title -> event title
	// each list is sorted by title.
	tds, err := cts.ReadDirectory(false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// this gets all values of all things. Maybe more efficient
	// to establish a shared cache?
	data := RenderSelectSourcesTemplateData{
		Values: make(map[string]hubclient.ThingMessageMap),
	}
	data.AgentThings = vm.GroupByAgent(tds)
	for thingID, _ := range tds {
		tm, _ := vm.GetLatest(thingID)
		data.Values[thingID] = tm
	}
	buff, err := app.RenderAppOrFragment(r, RenderSelectSourceTemplateFile, data)
	sess.WritePage(w, buff, err)
}
