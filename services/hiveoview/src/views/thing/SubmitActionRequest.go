package thing

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"net/http"
)

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
			thingAddr := fmt.Sprintf("%s/%s", thingID, actionName)
			propVal := tputils.DecodeAsString(reply, 0)
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
	//sess.SendNotify(session.NotifyInfo, "Delivery Status for '"+actionName+"': "+stat.Status)
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
