package thing

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
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
	Input  *SchemaValue
	Output *SchemaValue
	// the message with the action
	Msg things.ThingMessage
	// current delivery status
	Status hubclient.DeliveryStatus
	// Previous input value
	PrevValue *things.ThingMessage
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
	var td *things.TD
	var actionAff *things.ActionAffordance
	var hc hubclient.IHubClient

	lastAction := &things.ThingMessage{}

	// Read the TD being displayed
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc = mySession.GetHubClient()
		td, actionAff, err = getActionAff(hc, thingID, key)
	}
	// last action that was submitted
	if err == nil {
		resp, err := digitwin.InboxReadLatest(hc, []string{key}, "", thingID)
		if err == nil {
			lastAction = resp.Get(key)
		}
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	data := ActionDialogData{
		ThingID:   thingID,
		Key:       key,
		TD:        td,
		Action:    actionAff,
		PrevValue: lastAction,
	}
	if actionAff.Input != nil {
		data.Input = &SchemaValue{
			ThingID:    thingID,
			Key:        key,
			DataSchema: actionAff.Input,
			Value:      lastAction.DataAsText(),
		}
	}
	if actionAff.Output != nil {
		data.Output = &SchemaValue{
			ThingID:    thingID,
			Key:        key,
			DataSchema: actionAff.Output,
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
	valueStr := r.FormValue(actionKey)
	newValue = valueStr

	stat := hubclient.DeliveryStatus{}
	//
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc = mySession.GetHubClient()

		// convert the value from string to the data type
		td, actionAff, err = getActionAff(hc, thingID, actionKey)
		_ = td
	}
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
		stat = hc.PubAction(thingID, actionKey, newValue)
		if stat.Error != "" {
			err = errors.New(stat.Error)
		}
	}
	if err != nil {
		slog.Warn("PostStartAction failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("thingID", thingID),
			slog.String("actionKey", actionKey),
			slog.String("err", err.Error()))

		// notify UI via SSE. This is handled by a toast component.
		mySession.SendNotify(session.NotifyError, err.Error())

		// todo, differentiate between server error, invalid value and unauthorized
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: map delivery status to language

	// the async reply will contain status update
	mySession.SendNotify(session.NotifyInfo, "Delivery Progress for '"+actionKey+"': "+stat.Progress)

	w.WriteHeader(http.StatusOK)

}
