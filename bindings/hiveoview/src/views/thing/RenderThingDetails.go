package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/history"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

const TemplateFile = "RenderThingDetails.gohtml"
const RenderActionRequestPath = "/action/{thingID}/{key}/request"
const RenderEditPropertyPath = "/property/{thingID}/{key}/edit"
const RenderConfirmDeleteTDPath = "/directory/{thingID}/confirmDelete"

type ThingDetailsTemplateData struct {
	Title      string
	AgentID    string
	ThingID    string
	MakeModel  string
	ThingName  string
	DeviceType string
	TD         *tdd.TD
	// split the properties in attributes and config for presentation
	AttrKeys   []string
	Attributes map[string]*tdd.PropertyAffordance
	ConfigKeys []string
	Config     map[string]*tdd.PropertyAffordance
	// latest value of properties
	Values hubclient.ThingMessageMap

	VM *session.ClientViewModel

	// URLs
	RenderConfirmDeleteTDPath string
}

// GetHistory returns the 24 hour history for the given key
func (dt *ThingDetailsTemplateData) GetHistory(key string) *history.HistoryTemplateData {
	timestamp := time.Now()
	hsd, err := history.NewHistoryTemplateData(dt.VM, dt.TD, key, timestamp, -24*3600)
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

// RenderThingDetails renders thing details view fragment 'thingDetails.html'
// URL parameters:
// @param thingID to view
func RenderThingDetails(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	agentID, _ := tdd.SplitDigiTwinThingID(thingID)

	pathParams := map[string]string{"thingID": thingID}
	thingData := &ThingDetailsTemplateData{
		Attributes:                make(map[string]*tdd.PropertyAffordance),
		AttrKeys:                  make([]string, 0),
		Config:                    make(map[string]*tdd.PropertyAffordance),
		ConfigKeys:                make([]string, 0),
		AgentID:                   agentID,
		ThingID:                   thingID,
		Title:                     "details of thing",
		RenderConfirmDeleteTDPath: utils.Substitute(RenderConfirmDeleteTDPath, pathParams),
	}

	// Read the TD being displayed and its latest values
	sess, hc, err := session.GetSessionFromContext(r)
	vm := sess.GetViewModel()
	if err == nil {
		thingData.VM = vm
		tdJson, err2 := digitwin.DirectoryReadTD(hc, thingID)
		td := tdd.TD{}
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
			propMap, err2 := vm.GetLatest(thingID)
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
