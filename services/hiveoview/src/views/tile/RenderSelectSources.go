package tile

import (
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/directory"
	"net/http"
)

const RenderSelectSourceTemplateFile = "RenderSelectSources.gohtml"

type RenderSelectSourcesTemplateData struct {
	ctDir       *consumedthing.ConsumedThingsDirectory
	AgentThings []*directory.AgentThings
	// map of thing latest event values
	//Values map[string]transports.IConsumerMap
	// Map of thingID to thing interaction affordances
	//IOValues map[string]consumedthing.InteractionOutputMap
}

// GetActionValue returns the action value of a tile source
func (data RenderSelectSourcesTemplateData) GetActionValue(thingID, name string) string {
	return data.GetValue(messaging.AffordanceTypeAction, thingID, name)
}

// GetActionUpdated returns the action timestamp of a tile source
func (data RenderSelectSourcesTemplateData) GetActionUpdated(thingID, name string) string {
	return data.GetUpdated(messaging.AffordanceTypeAction, thingID, name)
}

// GetEventValue returns the property value of a tile source
func (data RenderSelectSourcesTemplateData) GetEventValue(thingID, name string) string {
	return data.GetValue(messaging.AffordanceTypeEvent, thingID, name)
}

// GetEventUpdated returns the property timestamp of a tile source
func (data RenderSelectSourcesTemplateData) GetEventUpdated(thingID, name string) string {
	return data.GetUpdated(messaging.AffordanceTypeEvent, thingID, name)
}

// GetPropertyValue returns the property value of a tile source
func (data RenderSelectSourcesTemplateData) GetPropertyValue(thingID, name string) string {
	return data.GetValue(messaging.AffordanceTypeProperty, thingID, name)
}

// GetPropertyUpdated returns the property timestamp of a tile source
func (data RenderSelectSourcesTemplateData) GetPropertyUpdated(thingID, name string) string {
	return data.GetUpdated(messaging.AffordanceTypeProperty, thingID, name)
}

// GetValue returns the value of a tile source
func (data RenderSelectSourcesTemplateData) GetValue(affType, thingID, name string) string {
	ct, err := data.ctDir.Consume(thingID)
	if err != nil {
		// should never happen, but just in case
		return err.Error()
	}
	iout := ct.GetValue(affType, name)
	if iout != nil {
		unitSymbol := iout.UnitSymbol()
		return iout.Value.Text() + " " + unitSymbol
	}
	return ""
}
func (data RenderSelectSourcesTemplateData) GetUpdated(affType, thingID, name string) string {
	ct, err := data.ctDir.Consume(thingID)
	if err != nil {
		return ""
	}
	iout := ct.GetValue(affType, name)
	if iout != nil {
		tputils.DecodeAsDatetime(iout.Updated)
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
	buff, err := app.RenderAppOrFragment(r, RenderSelectSourceTemplateFile, data)

	// TODO: TBD Retarget to #modalLevel2 so the gohtml doesn't need to know
	sess.WritePage(w, buff, err)
}
