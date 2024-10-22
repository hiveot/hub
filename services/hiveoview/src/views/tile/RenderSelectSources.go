package tile

import (
	"github.com/hiveot/hub/api/go/digitwin"
	session2 "github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/wot/consumedthing"
	"net/http"
)

const RenderSelectSourceTemplateFile = "RenderSelectSources.gohtml"

type RenderSelectSourcesTemplateData struct {
	AgentThings []*session2.AgentThings
	// map of thing latest event values
	//Values map[string]hubclient.ThingMessageMap
	// Map of thingID to thing interaction affordances
	IOValues map[string]consumedthing.InteractionOutputMap
}

// GetUpdated returns the update timestamp of the latest event value
func (data RenderSelectSourcesTemplateData) GetUpdated(thingID string, key string) string {
	ioMap, found := data.IOValues[thingID]
	if !found {
		return ""
	}
	io, found := ioMap[key]
	if !found {
		return ""
	}
	return io.GetUpdated()
}

// GetValue returns the string value of a thing event
func (data RenderSelectSourcesTemplateData) GetValue(thingID string, key string) string {
	ioMap, found := data.IOValues[thingID]
	if !found {
		return ""
	}
	io, found := ioMap[key]
	if !found {
		return ""
	}
	return io.Value.Text()
}

// RenderSelectSources renders the selection of Tile sources for adding to a tile
// A source is either an event or action.
// TODO: split into properties, events and actions
func RenderSelectSources(w http.ResponseWriter, r *http.Request) {

	sess, hc, err := session2.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	vm := sess.GetViewModel()
	cts := sess.GetConsumedThingsDirectory()
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
		//Values: make(map[string]hubclient.ThingMessageMap),
		IOValues: make(map[string]consumedthing.InteractionOutputMap),
	}
	data.AgentThings = vm.GroupByAgent(tds)
	for thingID, td := range tds {
		//propValues, err := digitwin.ValuesReadAllProperties(hc, thingID)
		eventValues, err := digitwin.ValuesReadAllEvents(hc, thingID)
		if err == nil {
			data.IOValues[thingID] = consumedthing.NewInteractionOutputFromValueList(eventValues, td)
		}
		//tm, _ := vm.GetLatest(thingID)
		//data.Values[thingID] = tm
		//data.IOValues = consumedthing.NewInteractionOutputFromTM(tm, td)
	}
	buff, err := app.RenderAppOrFragment(r, RenderSelectSourceTemplateFile, data)

	// TODO: TBD Retarget to #modalLevel2 so the gohtml doesn't need to know
	sess.WritePage(w, buff, err)
}
