package tile

import (
	"net/http"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/lib/messaging"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/directory"
)

const RenderSelectSourceTemplate = "SelectSources.gohtml"

type RenderSelectSourcesTemplateData struct {
	ctDir       *consumedthing.ConsumedThingsDirectory
	AgentThings []*directory.AgentThings
	// map of thing latest event values
	//Values map[string]transports.IConsumerMap
	// Map of thingID to thing interaction affordances
	//IOValues map[string]consumedthing.InteractionOutputMap
}

// GetActionValue returns the action value of a source
func (data RenderSelectSourcesTemplateData) GetActionValue(thingID, name string) *consumedthing.InteractionOutput {
	return data.GetValue(messaging.AffordanceTypeAction, thingID, name)
}

// TODO: use interactionoutput from consumed thing?
//  this relates to the slow loading with lots of values. Is there a need to create
//  all consumed things just to pick a value to add to the tile?
//  or create dynamically?
//  tbd
//  alternatively, show only the schema and get the value on demand?

// GetEventValue returns the property value of a tile source
func (data RenderSelectSourcesTemplateData) GetEventValue(thingID, name string) *consumedthing.InteractionOutput {
	return data.GetValue(messaging.AffordanceTypeEvent, thingID, name)
}

// GetPropertyValue returns the property value of a tile source
func (data RenderSelectSourcesTemplateData) GetPropertyValue(thingID, name string) *consumedthing.InteractionOutput {
	return data.GetValue(messaging.AffordanceTypeProperty, thingID, name)
}

// GetValue returns the value of a tile source
func (data RenderSelectSourcesTemplateData) GetValue(
	affType messaging.AffordanceType, thingID, name string) *consumedthing.InteractionOutput {
	ct, err := data.ctDir.Consume(thingID)
	if err != nil {
		// should never happen, but just in case
		return consumedthing.NoValue()
	}
	iout := ct.GetValue(affType, name)
	return iout
}
func (data RenderSelectSourcesTemplateData) GetUpdated(affType messaging.AffordanceType, thingID, name string) string {
	ct, err := data.ctDir.Consume(thingID)
	if err != nil {
		return ""
	}
	iout := ct.GetValue(affType, name)
	if iout != nil {
		utils.FormatDateTime(iout.Timestamp)
	}
	return ""
}

// RenderSelectSources renders the selection of Tile sources for adding to a tile
// A source is a property or event or action.
func RenderSelectSources(w http.ResponseWriter, r *http.Request) {

	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	ctDir := sess.GetConsumedThingsDirectory()
	// provide a list of things for each agent and show a nested
	// list of events per thing: agent -> thing title -> event title
	// each list is sorted by title.
	tds, err := ctDir.ReadDirectory(false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// this gets all values of all things. Maybe more efficient
	// to establish a shared cache?
	data := RenderSelectSourcesTemplateData{
		ctDir: ctDir,
	}
	data.AgentThings = directory.GroupByAgent(tds)
	buff, err := app.RenderAppOrFragment(r, RenderSelectSourceTemplate, data)

	// TODO: TBD Retarget to #modalLevel2 so the gohtml doesn't need to know
	sess.WritePage(w, buff, err)
}
