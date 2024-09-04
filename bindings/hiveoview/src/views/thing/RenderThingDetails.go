package thing

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/history"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/consumedthing"
	"github.com/hiveot/hub/wot/tdd"
	"golang.org/x/exp/maps"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

const TemplateFile = "RenderThingDetails.gohtml"
const RenderActionRequestPath = "/action/{thingID}/{key}/request"
const RenderEditPropertyPath = "/property/{thingID}/{key}/edit"
const RenderConfirmDeleteTDPath = "/directory/{thingID}/confirmDeleteTD"
const RenderRawTDPath = "/thing/{thingID}/raw"

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
	ConfigKeys []string
	EventKeys  []string
	ActionKeys []string

	// latest value of properties
	Values map[string]*consumedthing.InteractionOutput
	CT     *consumedthing.ConsumedThing
	//VM *session.ClientViewModel

	// URLs
	RenderConfirmDeleteTDPath string
	RenderRawTDPath           string
}

// GetHistory returns the previous 24 hour for the given key
func (dt *ThingDetailsTemplateData) GetHistory(key string) *history.HistoryTemplateData {
	timestamp := time.Now()
	duration := time.Hour * time.Duration(-24)
	hsd, err := history.NewHistoryTemplateData(dt.CT, key, timestamp, duration)
	_ = err
	return hsd
}

// GetSenderID returns the last sender of a value
func (dt *ThingDetailsTemplateData) GetSenderID(key string) string {
	io, found := dt.Values[key]
	if !found {
		return ""
	}
	return io.GetSenderID()
}

// GetUpdated returns the timestamp the value was last updated
func (dt *ThingDetailsTemplateData) GetUpdated(key string) string {
	io, found := dt.Values[key]
	if !found {
		return ""
	}
	return io.GetUpdated()
}

// GetValue returns the interaction output of the last value of event or property
// If the value is unknown, a dummy value is returned to avoid crashing.
func (dt *ThingDetailsTemplateData) GetValue(key string) *consumedthing.InteractionOutput {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("PANIC RECOVERED", "key=", key)
		}
	}()
	io, _ := dt.CT.GetValue(key)
	//_ = found //
	return io
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
	var ct *consumedthing.ConsumedThing

	pathParams := map[string]string{"thingID": thingID}
	thingData := &ThingDetailsTemplateData{
		AttrKeys:                  make([]string, 0),
		ConfigKeys:                make([]string, 0),
		EventKeys:                 make([]string, 0),
		ActionKeys:                make([]string, 0),
		AgentID:                   agentID,
		ThingID:                   thingID,
		Title:                     "details of thing",
		RenderConfirmDeleteTDPath: utils.Substitute(RenderConfirmDeleteTDPath, pathParams),
		RenderRawTDPath:           utils.Substitute(RenderRawTDPath, pathParams),
	}

	// Read the TD being displayed and its latest values
	sess, _, err := session.GetSessionFromContext(r)
	cts := sess.GetConsumedThingsSession()
	if err == nil {
		ct, err = cts.Consume(thingID)
	}
	if err != nil {
		slog.Error("Failed loading Thing info",
			"thingID", thingID, "err", err.Error())
		sess.SendNotify(session.NotifyError, err.Error())
		sess.WriteError(w, err, 0)
		return
	}
	thingData.CT = ct

	// split properties into attributes and configuration and update the keys list
	td := thingData.CT.GetThingDescription()
	thingData.TD = td
	for k, prop := range td.Properties {
		if prop.ReadOnly {
			thingData.AttrKeys = append(thingData.AttrKeys, k)
			//thingData.Attributes[k] = prop
		} else {
			thingData.ConfigKeys = append(thingData.ConfigKeys, k)
			//thingData.Config[k] = prop
		}
	}
	thingData.EventKeys = maps.Keys(td.Events)
	thingData.ActionKeys = maps.Keys(td.Actions)

	// sort the keys by name for presentation
	sort.SliceStable(thingData.AttrKeys, func(i int, j int) bool {
		k1 := thingData.AttrKeys[i]
		prop1, _ := td.Properties[k1]
		k2 := thingData.AttrKeys[j]
		prop2, _ := td.Properties[k2]
		return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
	})
	sort.SliceStable(thingData.ConfigKeys, func(i int, j int) bool {
		k1 := thingData.ConfigKeys[i]
		prop1, _ := td.Properties[k1]
		k2 := thingData.ConfigKeys[j]
		prop2, _ := td.Properties[k2]
		return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
	})
	sort.SliceStable(thingData.EventKeys, func(i int, j int) bool {
		k1 := thingData.EventKeys[i]
		ev1, _ := td.Events[k1]
		k2 := thingData.EventKeys[j]
		ev2, _ := td.Events[k2]
		return strings.ToLower(ev1.Title) < strings.ToLower(ev2.Title)
	})
	sort.SliceStable(thingData.ActionKeys, func(i int, j int) bool {
		k1 := thingData.ActionKeys[i]
		act1, _ := td.Actions[k1]
		k2 := thingData.ActionKeys[j]
		act2, _ := td.Actions[k2]
		return strings.ToLower(act1.Title) < strings.ToLower(act2.Title)
	})

	// get the latest event/property values from the outbox
	//propMap, err2 := vm.GetLatest(thingID)
	//err = err2
	propMap := thingData.CT.ReadAllProperties()
	thingData.Values = propMap
	thingData.DeviceType = td.AtType

	// get the value of a make & model properties, if they exist
	// TODO: this is a bit of a pain to do. Is this a common problem?
	makeID, _ := td.GetPropertyOfType(vocab.PropDeviceMake)
	modelID, _ := td.GetPropertyOfType(vocab.PropDeviceModel)
	//makeValue := propMap.Get(makeID)
	//modelValue := propMap.Get(modelID)
	//if makeValue != nil {
	//	thingData.MakeModel = makeValue.DataAsText() + ", "
	//}
	makeValue, found := propMap[makeID]
	if found {
		thingData.MakeModel = makeValue.ToString() + ", "
	}
	modelValue, found := propMap[modelID]
	if found {
		thingData.MakeModel = thingData.MakeModel + modelValue.ToString()
	}
	// use name from configuration if available. Fall back to title.
	//thingData.ThingName = thingData.Values.ToString(vocab.PropDeviceTitle)
	propID, _ := td.GetPropertyOfType(vocab.PropDeviceTitle)
	deviceTitleValue, found := propMap[propID]
	if found {
		thingData.ThingName = deviceTitleValue.ToString()
	}
	if thingData.ThingName == "" {
		thingData.ThingName = td.Title
	}
	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, TemplateFile, thingData)
	sess.WritePage(w, buff, err)
}
