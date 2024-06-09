package thing

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/digitwin/digitwinclient"
	"log/slog"
	"net/http"
)

const TemplateFile = "thingDetails.gohtml"

type DetailsTemplateData struct {
	AgentID    string
	ThingID    string
	MakeModel  string
	Name       string
	DeviceType string
	TD         *things.TD
	// These lists are sorted by property/event/action name
	Attributes map[string]*things.PropertyAffordance
	Config     map[string]*things.PropertyAffordance
	Values     things.ThingMessageMap
}

// return a map with the latest property values of a thing or nil if failed
func getLatest(thingID string, hc hubclient.IHubClient) (things.ThingMessageMap, error) {
	data := things.NewThingMessageMap()
	tvs, err := digitwinclient.ReadOutbox(hc, thingID)
	if err != nil {
		return data, err
	}
	for _, tv := range tvs {
		data.Set(tv.Key, tv)
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

	// Read the TD being displayed and its latest values
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		td, err2 := digitwinclient.ReadTD(hc, thingID)
		thingData.TD = td
		err = err2
		if err == nil {
			// split properties into attributes and configuration
			for k, prop := range thingData.TD.Properties {
				if prop.ReadOnly {
					thingData.Attributes[k] = prop
				} else {
					thingData.Config[k] = prop
				}
			}

			// get the latest values if available
			propMap, err2 := getLatest(thingID, hc)
			err = err2
			thingData.Values = propMap
			thingData.DeviceType = thingData.TD.AtType

			// get the value of a make & model properties, if they exist
			// TODO: this is a bit of a pain to do. Is this a common problem?
			makeID, _ := thingData.TD.GetPropertyOfType(vocab.PropDeviceMake)
			modelID, _ := thingData.TD.GetPropertyOfType(vocab.PropDeviceModel)
			makeValue := propMap.Get(makeID)
			modelValue := propMap.Get(modelID)
			if makeValue != nil {
				thingData.MakeModel = makeValue.DataAsText() + ", "
			}
			if modelValue != nil {
				thingData.MakeModel = thingData.MakeModel + modelValue.DataAsText()
			}
			// use name from configuration if available. Fall back to title.
			thingData.Name = thingData.Values.ToString(vocab.PropDeviceTitle)
			if thingData.Name == "" {
				thingData.Name = thingData.TD.Title
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
