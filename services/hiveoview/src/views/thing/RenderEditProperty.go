package thing

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/runtime/consumedthing"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"net/http"
)

const RenderEditPropertyTemplate = "RenderEditProperty.gohtml"

// Edit Thing Property data
type RenderEditPropertyTemplateData struct {
	ThingID    string
	Name       string
	DataSchema *td.DataSchema
	//Value      string
	// The last known value of the property to edit
	PropertyValue      consumedthing.InteractionOutput
	PropertyInput      consumedthing.InteractionInput
	SubmitPropertyPath string
}

//
//func getPropAff(hc transports.IConsumerConnection, thingID string, name string) (
//	td *td.TD, propAff *td.PropertyAffordance, err error) {
//
//	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
//	if err != nil {
//		return td, propAff, err
//	}
//	err = jsoniter.UnmarshalFromString(tdJson, &td)
//	if err != nil {
//		return td, propAff, err
//	}
//	propAff = td.GetProperty(name)
//	if propAff == nil {
//		return td, propAff, fmt.Errorf("property '%s' not found for Thing '%s'", name, thingID)
//	}
//	return td, propAff, nil
//}

// obtain a schema value instance from a thingID and key
func getConfigValue(
	sess *session.WebClientSession, thingID string, name string) (sv RenderEditPropertyTemplateData, err error) {

	ct, err := sess.Consume(thingID)
	if err != nil {
		return sv, err
	}
	iout := ct.GetPropOutput(name)
	if iout == nil {
		return sv, errors.New("No such property: " + name)
	}
	iin := ct.GetPropInput(name)

	// FIXME: Readproperty - handled by whom? is the digitwin service name relevant
	// What does the wot specification for directory say?
	//propValue, err := digitwin.ThingValuesReadProperty(hc, name, thingID)
	//io := consumedthing.NewInteractionOutputFromValue(&propValue, td)
	sv = RenderEditPropertyTemplateData{
		//TD:         &td,
		ThingID:    thingID,
		Name:       name,
		DataSchema: &iout.Schema,
		//Value:         iout.Value.Text(),
		PropertyValue: *iout,
		PropertyInput: *iin,
	}
	return sv, nil
}

// RenderEditProperty renders the view for Thing property configuration
// This sets the data properties for AgentID, ThingID, Name and Config
func RenderEditProperty(w http.ResponseWriter, r *http.Request) {
	var data RenderEditPropertyTemplateData

	thingID := chi.URLParam(r, "thingID")
	propName := chi.URLParam(r, "name")

	_, sess, err := session.GetSessionFromContext(r)
	if err == nil {
		data, err = getConfigValue(sess, thingID, propName)
	}
	pathParams := map[string]string{"thingID": thingID, "name": propName}
	data.SubmitPropertyPath = tputils.Substitute(src.PostThingPropertyEditPath, pathParams)
	if err != nil {
		slog.Error("RenderEditConfigDialog:", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buff, err := app.RenderAppOrFragment(r, RenderEditPropertyTemplate, data)
	sess.WritePage(w, buff, err)
}
