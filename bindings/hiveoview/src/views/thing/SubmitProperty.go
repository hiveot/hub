package thing

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
)

// SubmitProperty handles posting of a new thing property value
// The posted form value contains a 'value' field
// TODO: use the form method from the TD - once forms are added
func SubmitProperty(w http.ResponseWriter, r *http.Request) {
	var newValue any
	var td *things.TD
	var propAff *things.PropertyAffordance
	var hc hubclient.IHubClient
	stat := hubclient.DeliveryStatus{}
	thingID := chi.URLParam(r, "thingID")
	propKey := chi.URLParam(r, "key")
	valueStr := r.FormValue(propKey)

	mySession, hc, err := session.GetSessionFromContext(r)
	if err == nil {
		td, propAff, err = getPropAff(hc, thingID, propKey)
		_ = td
	}
	slog.Info("Updating config",
		slog.String("thingID", thingID),
		slog.String("propKey", propKey),
		slog.String("value", valueStr))

	// form values are strings. Convert to their native type before posting
	if err == nil {
		newValue, err = things.ConvertToNative(valueStr, &propAff.DataSchema)

		// don't make this an rpc as the response time isn't always known with sleeping devices
		stat = hc.PubProperty(thingID, propKey, newValue)
		if stat.Error != "" {
			err = errors.New(stat.Error)
		}
	}
	if err != nil {
		slog.Warn("SubmitProperty failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("thingID", thingID),
			slog.String("propKey", propKey),
			slog.String("err", err.Error()))

		// todo, differentiate between error causes, eg 500 server error, 503 service not available, 400 invalid value and 401 unauthorized
		mySession.WriteError(w, err, http.StatusServiceUnavailable)
		return
	}

	// the async reply will contain status update
	w.WriteHeader(http.StatusOK)
}
