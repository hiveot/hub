package thing

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/http"
)

// SubmitProperty handles posting of a new thing property value
// The posted form value contains a 'value' field
// TODO: use the form method from the TD - once forms are added
func SubmitProperty(w http.ResponseWriter, r *http.Request) {
	var newValue any
	var td *tdd.TD
	var propAff *tdd.PropertyAffordance
	stat := hubclient.RequestProgress{}
	thingID := chi.URLParam(r, "thingID")
	propName := chi.URLParam(r, "name")
	valueStr := r.FormValue(propName)

	_, sess, err := session.GetSessionFromContext(r)
	if err == nil {
		td, propAff, err = getPropAff(sess.GetHubClient(), thingID, propName)
		_ = td
	}
	slog.Info("Updating config",
		slog.String("thingID", thingID),
		slog.String("propName", propName),
		slog.String("value", valueStr))

	// form values are strings. Convert to their native type before posting
	if err == nil {
		newValue, err = tdd.ConvertToNative(valueStr, &propAff.DataSchema)

		stat = sess.GetHubClient().WriteProperty(thingID, propName, newValue)
		if stat.Error != "" {
			err = errors.New(stat.Error)
		}
	}
	if err != nil {
		sess.SendNotify(session.NotifyError, "Property update failed: "+err.Error())

		slog.Warn("SubmitProperty failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("thingID", thingID),
			slog.String("propName", propName),
			slog.String("err", err.Error()))

		// todo, differentiate between error causes, eg 500 server error, 503 service not available, 400 invalid value and 401 unauthorized
		sess.WriteError(w, err, http.StatusServiceUnavailable)
		return
	}

	if stat.Progress == vocab.RequestCompleted {
		notificationText := fmt.Sprintf("Configuration changed.")
		sess.SendNotify(session.NotifySuccess, notificationText)
	} else {
		notificationText := fmt.Sprintf("Configuration request sent.")
		sess.SendNotify(session.NotifyInfo, notificationText)
	}

	// the async reply will contain status update
	w.WriteHeader(http.StatusOK)
}
