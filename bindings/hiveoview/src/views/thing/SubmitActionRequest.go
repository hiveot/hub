package thing

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/http"
)

// SubmitActionRequest posts the request to start an action
func SubmitActionRequest(w http.ResponseWriter, r *http.Request) {
	var td *tdd.TD
	var actionAff *tdd.ActionAffordance
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
			newValue, err = tdd.ConvertToNative(valueStr, actionAff.Input)
		}
	}
	if err == nil {
		slog.Info("SubmitActionRequest starting",
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
		slog.Warn("SubmitActionRequest failed",
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
