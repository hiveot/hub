package tile

import (
	"net/http"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/directory"
)

const RenderSelectSourceTemplate = "SelectSources.gohtml"

type RenderSelectSourcesTemplateData struct {
	// ctDir       *consumer.ConsumedThingsDirectory
	AgentThings []*directory.AgentThings
	// map of thing latest event values
	//Values map[string]transports.IConsumerMap
	// Map of thingID to thing interaction affordances
	//IOValues map[string]consumer.InteractionOutputMap
	sess *session.WebClientSession
}

// GetActionValue returns the action value of a source
func (templateData RenderSelectSourcesTemplateData) GetActionValue(
	thingID, name string) *consumer.InteractionOutput {

	return templateData.GetValue(thingID, msg.AffordanceTypeAction, name)
}

// TODO: use interactionoutput from consumed thing?
//  this relates to the slow loading with lots of values. Is there a need to create
//  all consumed things just to pick a value to add to the tile?
//  or create dynamically?
//  tbd
//  alternatively, show only the schema and get the value on demand?

// GetEventValue returns the property value of a tile source
func (templateData RenderSelectSourcesTemplateData) GetEventValue(
	thingID, name string) *consumer.InteractionOutput {

	return templateData.GetValue(thingID, msg.AffordanceTypeEvent, name)
}

// GetPropertyValue returns the property value of a tile source
func (templateData RenderSelectSourcesTemplateData) GetPropertyValue(
	thingID, name string) *consumer.InteractionOutput {

	return templateData.GetValue(thingID, msg.AffordanceTypeProperty, name)
}

// GetValue returns the value of a tile source
func (templateData RenderSelectSourcesTemplateData) GetValue(
	thingID string, affType msg.AffordanceType, name string) *consumer.InteractionOutput {

	ct, err := templateData.sess.GetConsumedThing(thingID)
	if err != nil {
		// should never happen, but just in case
		return consumer.NoValue()
	}
	iout := ct.GetIOutput(affType, name)
	return iout
}

func (templateData RenderSelectSourcesTemplateData) GetUpdated(
	affType msg.AffordanceType, thingID, name string) string {

	ct, err := templateData.sess.GetConsumedThing(thingID)
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
	// provide a list of things for each agent and show a nested
	// list of events per thing: agent -> thing title -> event title
	// each list is sorted by title.
	dirCl := sess.GetDirectory()
	tds, _ := dirCl.RetrieveAllThings(0, directory.DefaultMaxThings)

	// if err != nil {
	// 	sess.WriteError(w, err, http.StatusBadRequest)
	// 	return
	// }
	// this gets all values of all things. Maybe more efficient
	// to establish a shared cache?
	data := RenderSelectSourcesTemplateData{
		sess: sess,
	}
	data.AgentThings = directory.GroupByAgent(tds)
	buff, err := app.RenderAppOrFragment(r, RenderSelectSourceTemplate, data)

	// TODO: TBD Re-target to #modalLevel2 so the gohtml doesn't need to know
	sess.WritePage(w, buff, err)
}
