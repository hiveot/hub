package tile

import (
	"github.com/hiveot/hub/runtime/consumedthing"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/directory"
	"github.com/hiveot/hub/transports"
	"github.com/hiveot/hub/transports/tputils"
	"net/http"
)

const RenderSelectSourceTemplateFile = "RenderSelectSources.gohtml"

type RenderSelectSourcesTemplateData struct {
	AgentThings []*directory.AgentThings
	// map of thing latest event values
	//Values map[string]transports.IConsumerMap
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
	return tputils.DecodeAsDatetime(io.Updated)
}

// GetValue returns the string value and unit symbol of a thing event
// this is called from the template.
func (data RenderSelectSourcesTemplateData) GetValue(thingID string, key string) string {
	ioMap, found := data.IOValues[thingID]
	if !found {
		return ""
	}
	io, found := ioMap[key]
	if !found {
		return ""
	}
	valueStr := io.Value.Text()
	if io.UnitSymbol() != "" {
		valueStr += " " + io.UnitSymbol()
	}
	return valueStr
}

// RenderSelectSources renders the selection of Tile sources for adding to a tile
// A source is a property or event or action.
func RenderSelectSources(w http.ResponseWriter, r *http.Request) {

	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
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
		//Values: make(map[string]transports.IConsumerMap),
		IOValues: make(map[string]consumedthing.InteractionOutputMap),
	}
	data.AgentThings = directory.GroupByAgent(tds)
	for thingID, td := range tds {
		propMap, _ := sess.GetConsumer().ReadAllProperties(thingID)
		evMap, _ := sess.GetConsumer().ReadAllEvents(thingID)
		//propValues, err := digitwin.ThingValuesReadAllProperties(sess.GetConsumer(), thingID)
		if propMap != nil && evMap != nil {
			allValues := make([]transports.ThingValue, 0, len(propMap)+len(evMap))
			for _, v := range propMap {
				allValues = append(allValues, v)
			}
			for _, v := range evMap {
				allValues = append(allValues, v)
			}
			//eventValues, _ := digitwin.ThingValuesReadAllEvents(sess.GetConsumer(), thingID)
			//allValues := append(propValues, eventValues...)
			data.IOValues[thingID] = consumedthing.NewInteractionOutputFromValueList(
				td, transports.AffordanceTypeEvent, allValues)
		}
	}
	buff, err := app.RenderAppOrFragment(r, RenderSelectSourceTemplateFile, data)

	// TODO: TBD Retarget to #modalLevel2 so the gohtml doesn't need to know
	sess.WritePage(w, buff, err)
}
