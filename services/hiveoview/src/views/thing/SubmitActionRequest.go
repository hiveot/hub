package thing

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/hubclient"
	session2 "github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/http"
)

// SubmitActionRequest posts the request to start an action
func SubmitActionRequest(w http.ResponseWriter, r *http.Request) {
	var td *tdd.TD
	var actionAff *tdd.ActionAffordance
	var newValue any
	actionTitle := ""

	thingID := chi.URLParam(r, "thingID")
	actionName := chi.URLParam(r, "name")
	// booleans from form are non-values. Treat as false
	valueStr := r.FormValue(actionName)
	newValue = valueStr
	reply := ""

	stat := hubclient.RequestProgress{}
	//
	_, sess, err := session2.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
	}

	// convert the value from string to the data type
	td, actionAff, err = getActionAff(sess.GetHubClient(), thingID, actionName)
	if err != nil || td == nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	if actionAff.Input != nil {
		newValue, err = tdd.ConvertToNative(valueStr, actionAff.Input)
	}
	if err == nil {
		slog.Info("SubmitActionRequest starting",
			slog.String("thingID", thingID),
			slog.String("actionName", actionName),
			slog.Any("newValue", newValue))

		// FIXME: use async progress updates instead of RPC
		//stat = hc.HandleActionFlow(thingID, actionName, newValue)
		var resp interface{}
		err = sess.GetHubClient().Rpc(thingID, actionName, newValue, &resp)
		if stat.Error != "" {
			err = errors.New(stat.Error)
		} else if resp != nil {
			// stringify the reply for presenting in the notification
			reply = fmt.Sprintf("%v", resp)
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
		aff := td.GetAction(actionName)
		if aff != nil {
			actionTitle = aff.Title
		}
		if actionTitle == "" {
			actionTitle = actionName
		}

		err = fmt.Errorf("action '%s' of Thing '%s' failed: %w", actionTitle, td.Title, err)
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	} else {
		actionTitle = actionAff.Title
	}

	// TODO: map delivery status to language

	// the async reply will contain status update
	//sess.SendNotify(session.NotifyInfo, "Delivery Progress for '"+actionName+"': "+stat.Progress)
	unit := ""
	if actionAff.Output != nil {
		unit = actionAff.Output.Unit
	}
	notificationText := fmt.Sprintf("Action %s: %v %s", actionTitle, reply, unit)
	sess.SendNotify(session2.NotifySuccess, notificationText)

	w.WriteHeader(http.StatusOK)
}
