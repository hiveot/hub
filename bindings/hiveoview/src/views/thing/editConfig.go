package thing

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/digitwin/digitwinclient"
	"log/slog"
	"net/http"
)

// RenderEditThingConfig renders the view for editing Thing configuration property
// This sets the data properties for AgentID, ThingID, Key and Config
func RenderEditThingConfig(w http.ResponseWriter, r *http.Request) {
	var prop *things.PropertyAffordance
	var value string
	data := make(map[string]any)
	agentID := r.URL.Query().Get("agentID")
	thingID := r.URL.Query().Get("thingID")
	propKey := r.URL.Query().Get("key")

	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		dcl := digitwinclient.NewDirectoryClient(hc)
		//rd := dirclient.NewReadDirectoryClient(hc)
		td, err := dcl.ReadTD(thingID)
		if err == nil {
			prop = td.GetProperty(propKey)
		}
		if err == nil {
			ocl := digitwinclient.NewOutboxClient(hc)
			eventValues, _ := ocl.ReadLatest(thingID)
			if eventValues != nil && len(eventValues) > 0 {
				value = string(eventValues.ToString(propKey))
			}
		}
	}
	data["AgentID"] = agentID
	data["ThingID"] = thingID
	data["Key"] = propKey
	data["Config"] = prop
	data["Value"] = value

	app.RenderAppOrFragment(w, r, "editConfig.gohtml", data)
}

// PostThingConfig handles posting of a thing configuration update
// URL parameters:
// * agentID
// * thingID
// * key
// The posted form value contains a 'value' field
func PostThingConfig(w http.ResponseWriter, r *http.Request) {
	//agentID := chi.URLParam(r, "agentID")
	thingID := chi.URLParam(r, "thingID")
	propKey := chi.URLParam(r, "propKey")
	value := r.FormValue("value")
	stat := hubclient.DeliveryStatus{}
	//
	mySession, err := session.GetSessionFromContext(r)
	hc := mySession.GetHubClient()
	if err == nil {
		slog.Info("Updating config",
			slog.String("thingID", thingID),
			slog.String("propKey", propKey),
			slog.String("value", value))

		// don't make this an rpc as the response time isn't always known with sleeping devices
		stat = hc.PubAction(thingID, propKey, []byte(value))
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
	//mySession.SendNotify(session.NotifyInfo, "Delivery Status for '"+propKey+"': "+stat.Status)

	w.WriteHeader(http.StatusOK)

}
