package thing

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/comps"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"net/http"
	"time"
)

// ActionDialogData with data for the action window
type ActionDialogData struct {
	// The thing the action belongs to
	ThingID string
	TD      *things.TD
	// key of action
	Key string
	// the action being viewed in case of an action
	Action *things.ActionAffordance
	Input  *comps.SchemaValue
	Output *comps.SchemaValue
	// the message with the action
	Msg things.ThingMessage
	// current delivery status
	Status hubclient.DeliveryStatus
	// Previous input value
	PrevValue *digitwin.InboxRecord
	// timestamp the action was last performed
	LastUpdated string
	// duration the action was last performed
	LastUpdatedAge string
}

// Return the action affordance
func getActionAff(hc hubclient.IHubClient, thingID string, key string) (
	td *things.TD, actionAff *things.ActionAffordance, err error) {

	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
	if err != nil {
		return td, actionAff, err
	}
	err = json.Unmarshal([]byte(tdJson), &td)
	if err != nil {
		return td, actionAff, err
	}
	actionAff = td.GetAction(key)
	if actionAff == nil {
		return td, actionAff, fmt.Errorf("Action '%s' not found for Thing '%s'", key, thingID)
	}
	return td, actionAff, nil
}

// RenderActionDialog renders the action dialog.
// Path: /things/{thingID/{key}
//
//	@param thingID this is the URL parameter
func RenderActionDialog(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")
	var hc hubclient.IHubClient
	//var lastAction *digitwin.InboxRecord

	data := ActionDialogData{
		ThingID: thingID,
		Key:     key,
	}
	// Read the TD being displayed
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect?
		mySession.WriteError(w, err, http.StatusBadRequest)
		return
	}
	if err == nil {
		data.TD, data.Action, err = getActionAff(hc, thingID, key)
	}
	// last action that was submitted
	if err == nil {
		// reading a latest value is optional
		lastActionRecord, err2 := digitwin.InboxReadLatest(hc, key, thingID)
		if err2 == nil {
			data.PrevValue = &lastActionRecord
			updatedTime, _ := dateparse.ParseAny(lastActionRecord.Updated)
			data.LastUpdated = updatedTime.Format(time.RFC1123)
			data.LastUpdatedAge = utils.Age(updatedTime)
		}
	}
	if err != nil {
		mySession.WriteError(w, err, http.StatusBadRequest)
		return
	}

	prevInputValue := ""
	if data.PrevValue != nil && data.PrevValue.Input != nil {
		prevInputValue = fmt.Sprintf("%v", data.PrevValue.Input)
	}
	if data.Action.Input != nil {
		data.Input = &comps.SchemaValue{
			ThingID:    thingID,
			Key:        key,
			DataSchema: data.Action.Input,
			Value:      prevInputValue,
		}
	}
	if data.Action.Output != nil {
		data.Output = &comps.SchemaValue{
			ThingID:    thingID,
			Key:        key,
			DataSchema: data.Action.Output,
			Value:      "",
		}
	}
	app.RenderAppOrFragment(w, r, "actionDialog.gohtml", data)
}

// RenderProgress renders the action progress status component.
// Path: /app/progress/{messageID}
//
//	@param thingID this is the URL parameter
func RenderProgress(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "messageID")
	//action := thing.ActionAffordance{}
	_ = messageID
}

// PostStartAction posts the request to start an action
func PostStartAction(w http.ResponseWriter, r *http.Request) {
	var td *things.TD
	var actionAff *things.ActionAffordance
	var newValue any
	var hc hubclient.IHubClient

	thingID := chi.URLParam(r, "thingID")
	actionKey := chi.URLParam(r, "key")
	// booleans from form are non-values. Treat as false
	valueStr := r.FormValue(actionKey)
	newValue = valueStr
	reply := ""

	stat := hubclient.DeliveryStatus{}
	//
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		mySession.WriteError(w, err, http.StatusBadRequest)
	}

	// convert the value from string to the data type
	td, actionAff, err = getActionAff(hc, thingID, actionKey)
	_ = td
	if err == nil {
		if actionAff.Input != nil {
			newValue, err = things.ConvertToNative(valueStr, actionAff.Input)
		}
	}
	if err == nil {
		slog.Info("PostStartAction starting",
			slog.String("thingID", thingID),
			slog.String("actionKey", actionKey),
			slog.Any("newValue", newValue))

		// don't make this an rpc as the response time isn't always known with sleeping devices
		//stat = hc.PubAction(thingID, actionKey, newValue)
		var resp interface{}
		err = hc.Rpc(thingID, actionKey, newValue, &resp)
		if stat.Error != "" {
			err = errors.New(stat.Error)
		} else if resp != nil {
			// stringify the reply for presenting in the notification
			reply = fmt.Sprintf("%v", resp)
		}
	}
	if err != nil {
		slog.Warn("PostStartAction failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("thingID", thingID),
			slog.String("actionKey", actionKey),
			slog.String("err", err.Error()))

		// notify UI via SSE. This is handled by a toast component.
		// todo, differentiate between server error, invalid value and unauthorized
		mySession.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	// TODO: map delivery status to language

	// the async reply will contain status update
	//mySession.SendNotify(session.NotifyInfo, "Delivery Progress for '"+actionKey+"': "+stat.Progress)
	unit := ""
	if actionAff.Output != nil {
		unit = actionAff.Output.Unit
	}
	notificationText := fmt.Sprintf("Action %s: %v %s", actionKey, reply, unit)
	mySession.SendNotify(session.NotifySuccess, notificationText)

	w.WriteHeader(http.StatusOK)

}
