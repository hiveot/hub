package thing

import (
	"encoding/json"
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
)

// RenderConfigEditDialog renders the view for editing Thing configuration property
// This sets the data properties for AgentID, ThingID, Key and Config
func RenderConfigEditDialog(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	propKey := chi.URLParam(r, "key")

	var propAff *things.PropertyAffordance
	var value string
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {

		hc := mySession.GetHubClient()
		tdJson, err := digitwin.DirectoryReadTD(hc, thingID)

		if err == nil {
			td := things.TD{}
			err = json.Unmarshal([]byte(tdJson), &td)
			propAff = td.GetProperty(propKey)
		}
		if err == nil {
			eventValues, err2 := digitwin.OutboxReadLatest(hc, nil, "", thingID)
			err = err2
			valueMap := things.ThingMessageMap{}
			_ = json.Unmarshal([]byte(eventValues), &valueMap)
			if err == nil && valueMap != nil && len(valueMap) > 0 {
				value = valueMap.ToString(propKey)
			}
		}
	}
	if err != nil {
		slog.Error("RenderEditConfigDialog:", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := SchemaValue{
		Key:        propKey,
		DataSchema: &propAff.DataSchema,
		//Value:      fmt.Sprintf("%v", value),
		Value: value,
	}
	app.RenderAppOrFragment(w, r, "configEdit.gohtml", data)
}

// PostThingConfig handles posting of a thing configuration update
// The posted form value contains a 'value' field
// TODO: use the form method from the TD - once forms are added
func PostThingConfig(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	propKey := chi.URLParam(r, "key")
	value := r.FormValue("value")
	stat := hubclient.DeliveryStatus{}
	//
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		slog.Info("Updating config",
			slog.String("thingID", thingID),
			slog.String("propKey", propKey),
			slog.String("value", value))

		// don't make this an rpc as the response time isn't always known with sleeping devices
		stat = hc.PubProperty(thingID, propKey, value)
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

		// todo, differentiate between server error, invalid value and unauthorized
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: map delivery status to language

	// the async reply will contain status update
	//mySession.SendNotify(session.NotifyInfo, "Delivery Progress for '"+propKey+"': "+stat.Progress)

	w.WriteHeader(http.StatusOK)

}
