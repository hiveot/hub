package thing

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/gocore/utils"
	"github.com/hiveot/gocore/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/consumedthing"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
)

const RenderActionRequestTemplate = "ActionRequest.gohtml"

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
		updatedTime, _ := dateparse.ParseAny(data.LastActionRecord.TimeUpdated)
		data.LastActionTime = updatedTime.Format(time.RFC1123)
		data.LastActionAge = Age(updatedTime)
		data.InputValue.Value.Raw = data.LastActionRecord.Input
	}

	pathArgs := map[string]string{"thingID": data.ThingID, "name": data.Name}
	data.SubmitActionRequestPath = utils.Substitute(src.PostActionRequestPath, pathArgs)

	buff, err := app.RenderAppOrFragment(r, RenderActionRequestTemplate, data)
	sess.WritePage(w, buff, err)
}

// RenderActionStatus renders the action progress status component.
// TODO:
//
//	@param thingID this is the URL parameter
func RenderActionStatus(w http.ResponseWriter, r *http.Request) {
	correlationID := chi.URLParam(r, "correlationID")
	//action := thing.ActionAffordance{}
	_ = correlationID
}

// SubmitActionRequest posts the request to start an action
func SubmitActionRequest(w http.ResponseWriter, r *http.Request) {
	//var tdi *td.TD
	var newValue any
	actionTitle := ""

	// when RenderCardInput submits a checkbox its value is "" (false) or "on" (true)
	thingID := chi.URLParam(r, "thingID")
	actionName := chi.URLParam(r, "name")
	// booleans from form are non-values. Treat as false
	valueStr := r.FormValue(actionName)
	newValue = valueStr
	reply := ""

	//stat := transports.ActionStatus{}
	//
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
	}

	// convert the input value from string to the data type
	ct, err := sess.Consume(thingID)
	//tdi, actionAff, err = getActionAff(sess.GetConsumer(), thingID, actionName)
	if err != nil || ct == nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
		return
	}
	actionAff := ct.GetActionAff(actionName)
	if actionAff.Input != nil {
		newValue, err = td.ConvertToNative(valueStr, actionAff.Input)
	}
	if err == nil {
		slog.Info("SubmitActionRequest starting",
			slog.String("thingID", thingID),
			slog.String("actionName", actionName),
			slog.Any("newValue", newValue))

		// FIXME: use async progress updates instead of RPC
		//stat = hc.HandleActionFlow(thingID, actionName, newValue)
		var resp interface{}
		//iout,err = ct.InvokeAction(actionName, newValue, &resp)
		err = sess.GetConsumer().InvokeAction(thingID, actionName, newValue, &resp)
		if resp != nil {
			// stringify the reply for presenting in the notification
			reply = fmt.Sprintf("%v", resp)

			// Receiving a response means it isn't received by the async response handler,
			// so we need to let the UI know ourselves.
			// The browser is update with action updates using the
			//   property/thingID/propertyName SSE endpoint
			// See also the sse-swap attribute in gohtml templates
			thingAddr := fmt.Sprintf("property/%s/%s", thingID, actionName)
			propVal := utils.DecodeAsString(reply, 0)
			sess.SendSSE(thingAddr, propVal)
		}
	}
	if err != nil {
		slog.Warn("SubmitActionRequest error",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("thingID", thingID),
			slog.String("actionName", actionName),
			slog.String("err", err.Error()))

		// notify UI via SSE. This is handled by a toast component.
		// todo, differentiate between server error, invalid value and unauthorized
		// use human title from TD instead of action key to make error more presentable
		actionTitle = actionName
		if actionAff.Title != "" {
			actionTitle = actionAff.Title
		}

		err = fmt.Errorf("action '%s' of Thing '%s' failed: %w",
			actionTitle, ct.Title, err)
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	} else {
		actionTitle = actionAff.Title
	}

	// TODO: map delivery status to language

	// the async reply will contain status update
	//sess.SendNotify(session.NotifyInfo, "Delivery Status for '"+actionName+"': "+stat.State)
	unitSymbol := ""
	if actionAff.Output != nil {
		unit := actionAff.Output.Unit
		if unit != "" {
			unitInfo, found := vocab.UnitClassesMap[unit]
			if found {
				unitSymbol = unitInfo.Symbol
			}
		}
	}

	notificationText := fmt.Sprintf("Action %s: %v %s", actionTitle, reply, unitSymbol)
	sess.SendNotify(session.NotifySuccess, "", notificationText)

	w.WriteHeader(http.StatusOK)
}
