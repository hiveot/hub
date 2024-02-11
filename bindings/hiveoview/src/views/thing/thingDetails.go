package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/core/history/historyclient"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
)

const TemplateFile = "thingDetails.gohtml"

type DetailsTemplateData struct {
	AgentID string
	ThingID string
	TD      things.TD
	// These lists are sorted by property/event/action name
	Attributes map[string]*things.PropertyAffordance
	Config     map[string]*things.PropertyAffordance
	Values     things.ThingValueMap
}

// return a map with the latest property values of a thing or nil if failed
func getLatest(agentID string, thingID string, hc *hubclient.HubClient) (things.ThingValueMap, error) {
	data := things.NewThingValueMap()
	rh := historyclient.NewReadHistoryClient(hc)
	tvs, err := rh.GetLatest(agentID, thingID, nil)
	if err != nil {
		return data, err
	}
	for _, tv := range tvs {
		data.Set(tv.Name, tv)
		if tv.Data == nil {
			tv.Data = []byte("")
		}
	}
	//_ = data.of("")
	return data, nil
}

// RenderThingDetails renders thing details view fragment 'thingDetails.html'
// URL parameters:
// @param agentID of the publisher
// @param thingID to view
func RenderThingDetails(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]any)
	agentID := chi.URLParam(r, "agentID")
	thingID := chi.URLParam(r, "thingID")
	thingData := &DetailsTemplateData{
		Attributes: make(map[string]*things.PropertyAffordance),
		Config:     make(map[string]*things.PropertyAffordance),
	}
	thingData.ThingID = thingID
	thingData.AgentID = agentID
	data["Thing"] = thingData
	data["Title"] = "details of thing"

	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		rd := dirclient.NewReadDirectoryClient(hc)
		tv, err2 := rd.GetTD(agentID, thingID)
		err = err2
		if err == nil {
			err = json.Unmarshal(tv.Data, &thingData.TD)
			// split properties into attributes and configuration
			for k, prop := range thingData.TD.Properties {
				if prop.ReadOnly {
					thingData.Attributes[k] = prop
				} else {
					thingData.Config[k] = prop
				}
			}

			// get the latest values if available
			propMap, err2 := getLatest(agentID, thingID, hc)
			err = err2
			thingData.Values = propMap
		}
	}
	if err != nil {
		slog.Error("Failed loading Thing info",
			"agentID", agentID, "thingID", thingID, "err", err.Error())
	}
	// full render or fragment render
	app.RenderAppOrFragment(w, r, TemplateFile, data)
}
