package thing

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	thing "github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
)

// ActionDialogData with data for the action window
type ActionDialogData struct {
	// The thing the action belongs to
	ThingID string
	TD      *thing.TD
	// key of action
	Key string
	// the action being viewed in case of an action
	Action *thing.ActionAffordance
	Input  *IOSchema
	Output *IOSchema
	// the message with the action
	Msg thing.ThingMessage
	// Delivery status
	Status hubclient.DeliveryStatus
}

// RenderActionDialog renders the action dialog.
// Path: /things/{thingID/{key}
//
//	@param thingID this is the URL parameter
func RenderActionDialog(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")
	td := thing.TD{}
	//action := thing.ActionAffordance{}
	tdJson := ""

	// Read the TD being displayed
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		tdJson, err = digitwin.DirectoryReadTD(hc, thingID)
		if err == nil {
			err = json.Unmarshal([]byte(tdJson), &td)
		}
	}
	action, found := td.Actions[key]
	if !found {
		http.Error(w, "Unknown action: "+key, http.StatusBadRequest)
		return
	}
	data := ActionDialogData{
		ThingID: thingID,
		Key:     key,
		TD:      &td,
		Action:  action,
	}
	if action.Input != nil {
		data.Input = &IOSchema{
			ThingID:    thingID,
			Key:        key,
			DataSchema: action.Input,
			Value:      "",
		}
	}
	if action.Output != nil {
		data.Output = &IOSchema{
			ThingID:    thingID,
			Key:        key,
			DataSchema: action.Output,
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

func PostStartAction(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	actionKey := chi.URLParam(r, "key")
	value := r.FormValue("value")
	stat := hubclient.DeliveryStatus{}
	//
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		slog.Info("PostStartAction starting",
			slog.String("thingID", thingID),
			slog.String("actionKey", actionKey),
			slog.String("value", value))

		// don't make this an rpc as the response time isn't always known with sleeping devices
		stat = hc.PubAction(thingID, actionKey, value)
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
