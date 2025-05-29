package thing

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"net/http"
)

const RenderEditPropertyTemplate = "EditProperty.gohtml"

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
	iout := ct.GetPropertyOutput(name)
	if iout == nil {
		return sv, errors.New("No such property: " + name)
	}
	iin := ct.GetPropertyInput(name)

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

// SubmitProperty handles writing of a new thing property value
// The posted form value contains a 'value' field
func SubmitProperty(w http.ResponseWriter, r *http.Request) {
	var newValue any
	var ct *consumedthing.ConsumedThing
	var propAff *td.PropertyAffordance
	//stat := transports.ActionStatus{}
	thingID := chi.URLParam(r, "thingID")
	propName := chi.URLParam(r, "name")
	valueStr := r.FormValue(propName)

	slog.Info("Updating config",
		slog.String("thingID", thingID),
		slog.String("name", propName),
		slog.String("value", valueStr))

	_, sess, err := session.GetSessionFromContext(r)
	if err == nil {
		ct, err = sess.Consume(thingID)
		propAff = ct.GetPropertyAff(propName)
		if propAff == nil {
			err = fmt.Errorf("no such property '%s'", propName)
		}
	}
	// form values are strings. Convert to their native type before posting
	if err == nil && propAff != nil {
		newValue, err = td.ConvertToNative(valueStr, &propAff.DataSchema)

		// TODO: update progress while waiting for completion. Options
		// 1. provide callback that receives updates
		// 2. send request without waiting and update in onResponse

		err = sess.GetConsumer().WriteProperty(thingID, propName, newValue, true)
	}
	if err != nil {
		sess.SendNotify(session.NotifyError, "", "Property update failed: "+err.Error())

		slog.Warn("SubmitProperty failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("thingID", thingID),
			slog.String("propName", propName),
			slog.String("err", err.Error()))

		// todo, differentiate between error causes, eg 500 server error, 503 service not available, 400 invalid value and 401 unauthorized
		sess.WriteError(w, err, http.StatusServiceUnavailable)
		return
	}

	// TODO: update the notification if additional responses are received
	notificationText := fmt.Sprintf("Configuration '%s' applied.", propAff.Title)
	sess.SendNotify(session.NotifySuccess, "", notificationText)

	w.WriteHeader(http.StatusOK)
}
