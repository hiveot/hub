package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/core/history/historyclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
)

// RenderEditThingConfig renders the view for editing Thing configuration property
// This sets the data properties for AgentID, ThingID, Key and Config
func RenderEditThingConfig(w http.ResponseWriter, r *http.Request) {
	var prop *things.PropertyAffordance
	var td things.TD
	var value string
	data := make(map[string]any)
	agentID := r.URL.Query().Get("agentID")
	thingID := r.URL.Query().Get("thingID")
	propKey := r.URL.Query().Get("key")

	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		rd := dirclient.NewReadDirectoryClient(hc)
		tv, err := rd.GetTD(agentID, thingID)
		if err == nil {
			err = json.Unmarshal(tv.Data, &td)
			if err == nil {
				prop = td.GetProperty(propKey)
			}
		}
		if err == nil {
			rh := historyclient.NewReadHistoryClient(hc)
			tvs, _ := rh.GetLatest(agentID, thingID, []string{propKey})
			if tvs != nil && len(tvs) > 0 {
				value = string(tvs.ToString(propKey))
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
	agentID := chi.URLParam(r, "agentID")
	thingID := chi.URLParam(r, "thingID")
	propKey := chi.URLParam(r, "propKey")
	value := r.FormValue("value")
	//
	mySession, err := session.GetSessionFromContext(r)
	hc := mySession.GetHubClient()
	if err == nil {
		slog.Info("Updating config",
			"agentID", agentID, "thingID", thingID,
			"propKey", propKey, "value", value)
		err = hc.PubConfig(agentID, thingID, propKey, []byte(value))
	}
	if err != nil {
		slog.Warn("PostThingConfig failed",
			slog.String("remoteAddr", r.RemoteAddr),
			slog.String("agentID", agentID),
			slog.String("thingID", thingID),
			slog.String("propKey", propKey),
			slog.String("err", err.Error()))

		// notify UI via SSE. This is handled by a toast component.
		_ = mySession.SendSSE("notify", "error:"+err.Error())

		// todo, differentiate between server error, invalid value and unauthorized
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_ = mySession.SendSSE("notify", "success: Configuration '"+propKey+"' updated")

	w.WriteHeader(http.StatusOK)

}
