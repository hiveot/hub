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

func getPropAff(hc hubclient.IHubClient, thingID string, key string) (
	td *things.TD, propAff *things.PropertyAffordance, err error) {

	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
	if err != nil {
		return td, propAff, err
	}
	err = json.Unmarshal([]byte(tdJson), &td)
	if err != nil {
		return td, propAff, err
	}
	propAff = td.GetProperty(key)
	if propAff == nil {
		return td, propAff, fmt.Errorf("property '%s' not found for Thing '%s'", key, thingID)
	}
	return td, propAff, nil
}

// obtain a schema value instance from a thingID and key
// this pulls the TD from the server (todo: consider using a local cache)
func getConfigValue(
	hc hubclient.IHubClient, thingID string, key string) (sv SchemaValue, err error) {
	var found bool

	td, propAff, err := getPropAff(hc, thingID, key)
	_ = td

	keys := []string{key}
	propValues, err := digitwin.OutboxReadLatest(hc, keys, "", thingID)
	if err != nil {
		return sv, err
	}
	// convert the property value to string for presentation
	// TODO: make this simpler
	tmRaw, found := propValues[key]
	tm := things.ThingMessage{}
	tmJson, err := json.Marshal(tmRaw)
	_ = json.Unmarshal(tmJson, &tm)
	if !found {
		return sv, fmt.Errorf("value for thingID/property '%s/%s' not found", thingID, key)
	}
	valueStr := fmt.Sprintf("%v", tm.Data)
	sv = SchemaValue{
		//TD:         &td,
		ThingID:    thingID,
		Key:        key,
		DataSchema: &propAff.DataSchema,
		Value:      valueStr,
	}
	return sv, nil
}

// RenderConfigEditDialog renders the view for editing Thing configuration property
// This sets the data properties for AgentID, ThingID, Key and Config
func RenderConfigEditDialog(w http.ResponseWriter, r *http.Request) {
	var cv SchemaValue
	thingID := chi.URLParam(r, "thingID")
	propKey := chi.URLParam(r, "key")

	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		cv, err = getConfigValue(hc, thingID, propKey)
	}
	if err != nil {
		slog.Error("RenderEditConfigDialog:", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.RenderAppOrFragment(w, r, "configEdit.gohtml", cv)
}

// PostThingConfig handles posting of a thing configuration update
// The posted form value contains a 'value' field
// TODO: use the form method from the TD - once forms are added
func PostThingConfig(w http.ResponseWriter, r *http.Request) {
	var newValue any
	var td *things.TD
	var propAff *things.PropertyAffordance
	var hc hubclient.IHubClient
	stat := hubclient.DeliveryStatus{}
	thingID := chi.URLParam(r, "thingID")
	propKey := chi.URLParam(r, "key")
	valueStr := r.FormValue(propKey)

	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc = mySession.GetHubClient()
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
		slog.Warn("PostThingConfig failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("thingID", thingID),
			slog.String("propKey", propKey),
			slog.String("err", err.Error()))

		// notify UI via SSE. This is handled by a toast component.
		mySession.SendNotify(session.NotifyError, err.Error())

		// todo, differentiate between error causes, eg 500 server error, 503 service not available, 400 invalid value and 401 unauthorized
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	// the async reply will contain status update
	w.WriteHeader(http.StatusOK)

}
