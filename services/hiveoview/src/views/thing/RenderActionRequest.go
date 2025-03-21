package thing

import (
	"errors"
	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/messaging/tputils"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/wot/td"
	"net/http"
	"time"
)

const RenderActionRequestTemplate = "RenderActionRequest.gohtml"

// ActionRequestTemplateData with data for the action request view
type ActionRequestTemplateData struct {
	// The thing and name the action belongs to
	ThingID     string
	Name        string
	Description string

	// the thing instance used to apply the action
	CT *consumedthing.ConsumedThing

	// Affordance of the action to issue containing the input dataschema
	Action *td.ActionAffordance

	// input value to edit
	// This defaults to the last action input
	InputValue *consumedthing.InteractionInput

	// the previous action request record
	LastActionRecord *digitwin.ActionStatus
	// input value with previous action input (if any)
	//LastActionInput consumedthing.DataSchemaValue
	// previous action timestamp (formatted)
	LastActionTime string
	// previous action age (formatted)
	LastActionAge string

	//
	SubmitActionRequestPath string
}

//// Return the action affordance
//func getActionAff(hc transports.IConsumerConnection, thingID string, name string) (
//	td *td.TD, actionAff *td.ActionAffordance, err error) {
//
//	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
//	if err != nil {
//		return td, actionAff, err
//	}
//	err = jsoniter.UnmarshalFromString(tdJson, &td)
//	if err != nil {
//		return td, actionAff, err
//	}
//	actionAff = td.GetAction(name)
//	if actionAff == nil {
//		return td, actionAff, fmt.Errorf("Action '%s' not found for Thing '%s'", name, thingID)
//	}
//	return td, actionAff, nil
//}

// RenderActionRequest renders the action dialog.
// Path: /things/{thingID}/{name}
//
//	@param thingID this is the URL parameter
func RenderActionRequest(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	name := chi.URLParam(r, "name")
	//var lastAction *digitwin.InboxRecord

	// Read the TD being displayed
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	ct, err := sess.Consume(thingID)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
	}
	tdi := ct.TD()
	actionAff := ct.GetActionAff(name)
	//_, actionAff, err := getActionAff(sess.GetConsumer(), thingID, name)
	if actionAff == nil {
		sess.WriteError(w, errors.New("No such action: "+name), http.StatusBadRequest)
		return
	}
	data := ActionRequestTemplateData{
		ThingID: thingID,
		Name:    name,
		Action:  actionAff,
		CT:      ct,
		// The input value will be set to the last action value, if available
		InputValue:  consumedthing.NewInteractionInput(tdi, name, nil),
		Description: actionAff.Description,
	}
	if data.Description == "" {
		data.Description = actionAff.Title
	}

	// get last action request that was received
	// reading a latest value is optional
	actionVal, err := digitwin.ThingValuesQueryAction(sess.GetConsumer(), name, thingID)
	if err == nil && actionVal.Name != "" {
		data.LastActionRecord = &actionVal
		//data.PrevValue = &lastActionRecord
		updatedTime, _ := dateparse.ParseAny(data.LastActionRecord.Updated)
		data.LastActionTime = updatedTime.Format(time.RFC1123)
		data.LastActionAge = tputils.Age(updatedTime)
		data.InputValue.Value.Raw = data.LastActionRecord.Input
	}

	pathArgs := map[string]string{"thingID": data.ThingID, "name": data.Name}
	data.SubmitActionRequestPath = tputils.Substitute(src.PostActionRequestPath, pathArgs)

	buff, err := app.RenderAppOrFragment(r, RenderActionRequestTemplate, data)
	sess.WritePage(w, buff, err)
}

// RenderActionStatus renders the action progress status component.
// TODO
//
//	@param thingID this is the URL parameter
func RenderActionStatus(w http.ResponseWriter, r *http.Request) {
	correlationID := chi.URLParam(r, "correlationID")
	//action := thing.ActionAffordance{}
	_ = correlationID
}
