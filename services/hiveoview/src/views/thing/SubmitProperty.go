package thing

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"net/http"
)

// SubmitProperty handles writing of a new thing property value
// The posted form value contains a 'value' field
func SubmitProperty(w http.ResponseWriter, r *http.Request) {
	var newValue any
	var tdi *td.TD
	var propAff *td.PropertyAffordance
	//stat := transports.ActionStatus{}
	thingID := chi.URLParam(r, "thingID")
	propName := chi.URLParam(r, "name")
	valueStr := r.FormValue(propName)

	_, sess, err := session.GetSessionFromContext(r)
	if err == nil {
		tdi, propAff, err = getPropAff(sess.GetHubClient(), thingID, propName)
		_ = tdi
	}
	slog.Info("Updating config",
		slog.String("thingID", thingID),
		slog.String("name", propName),
		slog.String("value", valueStr))

	// form values are strings. Convert to their native type before posting
	if err == nil {
		newValue, err = td.ConvertToNative(valueStr, &propAff.DataSchema)

		// TODO: update progress while waiting for completion. Options
		// 1. provide callback that receives updates
		// 2. send request without waiting and update in onResponse

		err = sess.GetHubClient().WriteProperty(thingID, propName, newValue, true)
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
