package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
	"sort"
	"strings"
)

const TemplateFile = "details.gohtml"

type DetailsTemplateData struct {
	Title      string
	AgentID    string
	ThingID    string
	MakeModel  string
	ThingName  string
	DeviceType string
	TD         *things.TD
	// These lists are sorted by property/event/action name
	AttrKeys   []string
	Attributes map[string]*things.PropertyAffordance
	ConfigKeys []string
	Config     map[string]*things.PropertyAffordance
	Values     things.ThingMessageMap
}

// return a map with the latest property values of a thing or nil if failed
func getLatest(thingID string, hc hubclient.IHubClient) (things.ThingMessageMap, error) {
	data := things.NewThingMessageMap()
	tvsJson, err := digitwin.OutboxReadLatest(hc, nil, "", thingID)
	if err != nil {
		return data, err
	}
	tvs, _ := things.NewThingMessageMapFromSource(tvsJson)
	for _, tv := range tvs {
		data.Set(tv.Key, tv)
	}
	//_ = data.of("")
	return data, nil
}

// RenderThingDetails renders thing details view fragment 'thingDetails.html'
// URL parameters:
// @param thingID to view
func RenderThingDetails(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	agentID, _ := things.SplitDigiTwinThingID(thingID)
	thingData := &DetailsTemplateData{
		Attributes: make(map[string]*things.PropertyAffordance),
		AttrKeys:   make([]string, 0),
		Config:     make(map[string]*things.PropertyAffordance),
		ConfigKeys: make([]string, 0),
		AgentID:    agentID,
		ThingID:    thingID,
		Title:      "details of thing",
	}

	// Read the TD being displayed and its latest values
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		tdJson, err2 := digitwin.DirectoryReadTD(hc, thingID)
		td := things.TD{}
		_ = json.Unmarshal([]byte(tdJson), &td)
		thingData.TD = &td
		err = err2
		if err == nil {
			// split properties into attributes and configuration and update the keys list
			for k, prop := range thingData.TD.Properties {
				if prop.ReadOnly {
					thingData.AttrKeys = append(thingData.AttrKeys, k)
					thingData.Attributes[k] = prop
				} else {
					thingData.ConfigKeys = append(thingData.ConfigKeys, k)
					thingData.Config[k] = prop
				}
			}
			// sort the keys by name for presentation
			sort.SliceStable(thingData.AttrKeys, func(i int, j int) bool {
				k1 := thingData.AttrKeys[i]
				prop1, _ := thingData.Attributes[k1]
				k2 := thingData.AttrKeys[j]
				prop2, _ := thingData.Attributes[k2]
				return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
			})
			sort.SliceStable(thingData.ConfigKeys, func(i int, j int) bool {
				k1 := thingData.ConfigKeys[i]
				prop1, _ := thingData.Config[k1]
				k2 := thingData.ConfigKeys[j]
				prop2, _ := thingData.Config[k2]
				return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
			})

			// get the latest event/property values from the outbox
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
			//thingData.ThingName = thingData.Values.ToString(vocab.PropDeviceTitle)
			propID, _ := thingData.TD.GetPropertyOfType(vocab.PropDeviceTitle)
			thingData.ThingName = thingData.Values.ToString(propID)

			if thingData.ThingName == "" {
				thingData.ThingName = thingData.TD.Title
			}
		}
	}
	if err != nil {
		slog.Error("DeliveryFailed loading Thing info",
			"thingID", thingID, "err", err.Error())
		mySession.SendNotify(session.NotifyError, err.Error())
	}
	// full render or fragment render
	app.RenderAppOrFragment(w, r, TemplateFile, thingData)
}

func RenderTDRaw(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	var tdJSON string
	var tdPretty []byte
	// Read the TD being displayed and its latest values
	mySession, err := session.GetSessionFromContext(r)
	if err == nil {
		hc := mySession.GetHubClient()
		tdJSON, err = digitwin.DirectoryReadTD(hc, thingID)
		// re-marshal with pretty-print JSON
		var tdObj any
		_ = json.Unmarshal([]byte(tdJSON), &tdObj)
		tdPretty, _ = json.MarshalIndent(tdObj, "", "    ")
	}
	w.Write(tdPretty)
	w.WriteHeader(http.StatusOK)

}
