package thing

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	session2 "github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/history"
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

type ThingDetailsTemplateData struct {
	Title      string
	AgentID    string
	ThingID    string
	MakeModel  string
	ThingName  string
	DeviceType string
	TD         *tdd.TD
	// split the properties in attributes and config for presentation
	AttrNames   []string
	ConfigNames []string
	EventNames  []string
	ActionNames []string

	// latest value of events
	//EventValues map[string]*consumedthing.InteractionOutput
	//PropValues  map[string]*consumedthing.InteractionOutput
	CT *consumedthing.ConsumedThing
	//VM *session.ClientViewModel

	// URLs
	RenderConfirmDeleteTDPath string
	RenderRawTDPath           string
}

// GetHistory returns the previous 24 hour for the given name
func (dt *ThingDetailsTemplateData) GetHistory(name string) *history.HistoryTemplateData {
	timestamp := time.Now()
	duration := time.Hour * time.Duration(-24)
	hsd, err := history.NewHistoryTemplateData(dt.CT, name, timestamp, duration)
	_ = err
	return hsd
}

// GetSenderID returns the last sender of a value
//func (dt *ThingDetailsTemplateData) GetSenderID(name string) string {
//	io, found := dt.EventValues[name]
//	if !found {
//		return ""
//	}
//	return io.SenderID
//}

// GetUpdated returns the timestamp the value was last updated
//func (dt *ThingDetailsTemplateData) GetUpdated(name string) string {
//	io, found := dt.Values[name]
//	if !found {
//		return ""
//	}
//	return "Z" + io.Updated
//}

// GetEventValue returns the interaction output of the last value of event or property
// If the value is unknown, a dummy value is returned to avoid crashing.
//func (dt *ThingDetailsTemplateData) GetEventValue(name string) *consumedthing.InteractionOutput {
//	defer func() {
//		if r := recover(); r != nil {
//			slog.Error("PANIC RECOVERED", "name=", name)
//		}
//	}()
//	io, _ := dt.CT.GetEventValue(name)
//	//_ = found //
//	return io
//}

// GetRenderEditPropertyPath returns the URL path for editing a property
func (dt *ThingDetailsTemplateData) GetRenderEditPropertyPath(name string) string {
	pathArgs := map[string]string{"thingID": dt.ThingID, "name": name}
	return utils.Substitute(src.RenderThingPropertyEditPath, pathArgs)
}

// GetRenderActionPath returns the URL path for rendering an action request
func (dt *ThingDetailsTemplateData) GetRenderActionPath(name string) string {
	pathArgs := map[string]string{"thingID": dt.ThingID, "name": name}
	return utils.Substitute(src.RenderActionRequestPath, pathArgs)
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
		AttrNames:                 make([]string, 0),
		ConfigNames:               make([]string, 0),
		EventNames:                make([]string, 0),
		ActionNames:               make([]string, 0),
		AgentID:                   agentID,
		ThingID:                   thingID,
		Title:                     "details of thing",
		RenderConfirmDeleteTDPath: utils.Substitute(src.RenderThingConfirmDeletePath, pathParams),
		RenderRawTDPath:           utils.Substitute(src.RenderThingRawPath, pathParams),
	}

	// Read the TD being displayed and its latest values
	sess, _, err := session2.GetSessionFromContext(r)
	cts := sess.GetConsumedThingsSession()
	if err == nil {
		ct, err = cts.Consume(thingID)
	}
	if err != nil {
		slog.Error("Failed loading Thing info",
			"thingID", thingID, "err", err.Error())
		sess.SendNotify(session2.NotifyError, err.Error())
		sess.WriteError(w, err, 0)
		return
	}
	thingData.CT = ct

	// split properties into attributes and configuration and update the names list
	td := thingData.CT.GetThingDescription()
	thingData.TD = td
	for k, prop := range td.Properties {
		if prop.ReadOnly {
			thingData.AttrNames = append(thingData.AttrNames, k)
			//thingData.Attributes[k] = prop
		} else {
			thingData.ConfigNames = append(thingData.ConfigNames, k)
			//thingData.Config[k] = prop
		}
	}
	thingData.EventNames = maps.Keys(td.Events)
	thingData.ActionNames = maps.Keys(td.Actions)

	// sort the name by title for presentation
	sort.SliceStable(thingData.AttrNames, func(i int, j int) bool {
		k1 := thingData.AttrNames[i]
		prop1, _ := td.Properties[k1]
		k2 := thingData.AttrNames[j]
		prop2, _ := td.Properties[k2]
		return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
	})
	sort.SliceStable(thingData.ConfigNames, func(i int, j int) bool {
		k1 := thingData.ConfigNames[i]
		prop1, _ := td.Properties[k1]
		k2 := thingData.ConfigNames[j]
		prop2, _ := td.Properties[k2]
		return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
	})
	sort.SliceStable(thingData.EventNames, func(i int, j int) bool {
		k1 := thingData.EventNames[i]
		ev1, _ := td.Events[k1]
		k2 := thingData.EventNames[j]
		ev2, _ := td.Events[k2]
		return strings.ToLower(ev1.Title) < strings.ToLower(ev2.Title)
	})
	sort.SliceStable(thingData.ActionNames, func(i int, j int) bool {
		k1 := thingData.ActionNames[i]
		act1, _ := td.Actions[k1]
		k2 := thingData.ActionNames[j]
		act2, _ := td.Actions[k2]
		return strings.ToLower(act1.Title) < strings.ToLower(act2.Title)
	})

	// get the latest event/property/action values
	//propMap, err2 := vm.GetLatest(thingID)
	//err = err2
	// FIXME: properties and events can hold different values
	//eventValues := thingData.CT.ReadAllEvents()
	//propValues := thingData.CT.ReadAllProperties()
	//thingData.EventValues = eventValues
	//thingData.PropValues = propValues

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
	makeValue := ct.GetPropValue(makeID)
	if makeValue.Value.Text() != "" {
		thingData.MakeModel = makeValue.Value.Text() + ", "
	}
	modelValue := ct.GetPropValue(modelID)
	if modelValue.Value.Text() != "" {
		thingData.MakeModel = thingData.MakeModel + modelValue.Value.Text()
	}
	// use name from configuration if available. Fall back to title.
	//thingData.ThingName = thingData.Values.ToString(vocab.PropDeviceTitle)
	propID, _ := td.GetPropertyOfType(vocab.PropDeviceTitle)
	deviceTitleValue := ct.GetPropValue(propID)
	if deviceTitleValue.Value.Text() != "" {
		thingData.ThingName = deviceTitleValue.Value.Text()
	}
	if thingData.ThingName == "" {
		thingData.ThingName = td.Title
	}
	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, TemplateFile, thingData)
	sess.WritePage(w, buff, err)
}
