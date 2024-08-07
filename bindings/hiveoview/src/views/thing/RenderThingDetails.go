package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/comps"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

const TemplateFile = "RenderThingDetails.gohtml"
const RenderActionRequestPath = "/action/{thingID}/{key}/request"
const RenderEditPropertyPath = "/property/{thingID}/{key}/edit"
const RenderConfirmDeleteTDPath = "/directory/{thingID}/confirmDeleteTDDialog"

type ThingDetailsTemplateData struct {
	Title      string
	AgentID    string
	ThingID    string
	MakeModel  string
	ThingName  string
	DeviceType string
	TD         *things.TD
	// split the properties in attributes and config for presentation
	AttrKeys   []string
	Attributes map[string]*things.PropertyAffordance
	ConfigKeys []string
	Config     map[string]*things.PropertyAffordance
	// latest value of properties
	Values things.ThingMessageMap
	//
	hc hubclient.IHubClient

	// URLs
	RenderConfirmDeleteTDPath string
}

// obtain the 24 hour history for the given key
func (dt *ThingDetailsTemplateData) GetHistory(key string) *comps.HistoryTemplateData {
	timestamp := time.Now()
	hsd, err := comps.NewHistoryTemplateData(dt.hc, dt.TD, key, timestamp, -24*3600)
	_ = err
	return hsd
}

// GetRenderEditPropertyPath returns the URL path for editing a property
func (dt *ThingDetailsTemplateData) GetRenderEditPropertyPath(key string) string {
	pathArgs := map[string]string{"thingID": dt.ThingID, "key": key}
	return utils.Substitute(RenderEditPropertyPath, pathArgs)
}

// GetRenderActionPath returns the URL path for rendering an action request
func (dt *ThingDetailsTemplateData) GetRenderActionPath(key string) string {
	pathArgs := map[string]string{"thingID": dt.ThingID, "key": key}
	return utils.Substitute(RenderActionRequestPath, pathArgs)
}

// GetLatest returns a map with the latest property values of a thing or nil if failed
// TODO: The generated API doesnt know return types because WoT TD has no
// place to defined them. Find a better solution.
func GetLatest(thingID string, hc hubclient.IHubClient) (things.ThingMessageMap, error) {
	data := things.NewThingMessageMap()
	tvsJson, err := digitwin.OutboxReadLatest(hc, nil, "", "", thingID)
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

	pathParams := map[string]string{"thingID": thingID}
	thingData := &ThingDetailsTemplateData{
		Attributes:                make(map[string]*things.PropertyAffordance),
		AttrKeys:                  make([]string, 0),
		Config:                    make(map[string]*things.PropertyAffordance),
		ConfigKeys:                make([]string, 0),
		AgentID:                   agentID,
		ThingID:                   thingID,
		Title:                     "details of thing",
		RenderConfirmDeleteTDPath: utils.Substitute(RenderConfirmDeleteTDPath, pathParams),
	}

	// Read the TD being displayed and its latest values
	sess, hc, err := session.GetSessionFromContext(r)
	if err == nil {
		thingData.hc = hc
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
			propMap, err2 := GetLatest(thingID, hc)
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
		slog.Error("Failed loading Thing info",
			"thingID", thingID, "err", err.Error())
		sess.SendNotify(session.NotifyError, err.Error())
	}
	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, TemplateFile, thingData)
	sess.WritePage(w, buff, err)
}
