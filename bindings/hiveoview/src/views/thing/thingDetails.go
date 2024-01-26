package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
)

const htmlFile = "thingDetails.html"

type DetailsTemplateData struct {
	AgentID string
	ThingID string
	TD      things.TD
}

// RenderThingDetails renders thing details view fragment 'thingDetails.html'
// URL parameters:
// @param agentID of the publisher
// @param thingID to view
func RenderThingDetails(w http.ResponseWriter, r *http.Request) {
	data := make(map[string]any)
	thingID := chi.URLParam(r, "thingID")
	agentID := chi.URLParam(r, "agentID")
	thingData := &DetailsTemplateData{}
	data["Thing"] = thingData
	data["Title"] = "details of thing"
	thingData.ThingID = thingID
	thingData.AgentID = agentID

	mySession, err := session.GetSession(w, r)
	if err == nil {
		hc := mySession.GetHubClient()
		rd := dirclient.NewReadDirectoryClient(hc)
		tv, err := rd.GetTD(agentID, thingID)
		if err == nil {
			err = json.Unmarshal(tv.Data, &thingData.TD)
		}
	}
	if err != nil {
		slog.Error("Failed loading Thing info",
			"agentID", agentID, "thingID", thingID, "err", err.Error())
	}
	views.TM.RenderTemplate(w, r, htmlFile, &data)
	//http.Redirect(w, r, "#thing", http.StatusSeeOther)

}
