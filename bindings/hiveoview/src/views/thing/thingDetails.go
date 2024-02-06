package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/core/directory/dirclient"
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
		tv, err := rd.GetTD(agentID, thingID)
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
		}
	}
	if err != nil {
		slog.Error("Failed loading Thing info",
			"agentID", agentID, "thingID", thingID, "err", err.Error())
	}
	// full render or fragment render
	app.RenderAppOrFragment(w, r, TemplateFile, data)
}
