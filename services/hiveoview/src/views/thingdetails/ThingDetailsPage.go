package thingdetails

import (
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/gocore/messaging"
	"github.com/hiveot/gocore/utils"
	"github.com/hiveot/gocore/wot/td"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/services/hiveoview/src/views/history"
	"golang.org/x/exp/maps"
)

const RenderThingDetailsPageTemplate = "ThingDetailsPage.gohtml"

type ThingDetailsTemplateData struct {
	AgentID    string
	ThingID    string
	MakeModel  string
	DeviceType string

	// split the properties in attributes and config for presentation
	AttrNames   []string
	ConfigNames []string
	EventNames  []string
	ActionNames []string

	CT *consumedthing.ConsumedThing

	// URLs
	RenderConfirmDeleteTDPath string
	RenderRawTDPath           string
}

// GetEventHistory returns the previous 24 hour for the given event name
// HistoryTemplateData is also the InteractionAffordance of the latest value
func (dt *ThingDetailsTemplateData) GetEventHistory(name string) *history.HistoryTemplateData {
	timestamp := time.Now().Local()
	duration := time.Hour * time.Duration(-24)
	//iout := dt.CT.ReadEvent(name)
	iout := dt.CT.GetValue(messaging.AffordanceTypeEvent, name)

	hist := historyclient.NewReadHistoryClient(dt.CT.GetConsumer())
	values, itemsRemaining, err := hist.ReadHistory(
		iout.ThingID, iout.Name, timestamp, duration, 500)
	_ = itemsRemaining
	_ = err // ignore for now
	// iout holds the latest value
	// values contain historical values from range timestamp-duration to timestamp
	hsd, err := history.NewHistoryTemplateData(iout, values, timestamp, duration)
	_ = err
	return hsd
}

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
	agentID, _ := td.SplitDigiTwinThingID(thingID)
	var ct *consumedthing.ConsumedThing

	pathParams := map[string]string{"thingID": thingID}
	thingData := &ThingDetailsTemplateData{
		AttrNames:                 make([]string, 0),
		ConfigNames:               make([]string, 0),
		EventNames:                make([]string, 0),
		ActionNames:               make([]string, 0),
		AgentID:                   agentID,
		ThingID:                   thingID,
		RenderConfirmDeleteTDPath: utils.Substitute(src.RenderThingConfirmDeletePath, pathParams),
		RenderRawTDPath:           utils.Substitute(src.RenderThingRawPath, pathParams),
	}

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		slog.Error("Missing session", "thingID", thingID, "err", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	cts := sess.GetConsumedThingsDirectory()
	ct, err = cts.Consume(thingID)
	if err != nil {
		slog.Error("Failed loading Thing info",
			"thingID", thingID, "err", err.Error())
		sess.SendNotify(session.NotifyError, "", err.Error())
		sess.WriteError(w, err, 0)
		return
	}
	thingData.CT = ct

	// split properties into attributes and configuration and update the names list
	tdi := thingData.CT.TD()

	for k, prop := range tdi.Properties {
		if prop.ReadOnly {
			thingData.AttrNames = append(thingData.AttrNames, k)
			//thingData.Attributes[k] = prop
		} else {
			thingData.ConfigNames = append(thingData.ConfigNames, k)
			//thingData.Config[k] = prop
		}
	}
	thingData.EventNames = maps.Keys(tdi.Events)
	thingData.ActionNames = maps.Keys(tdi.Actions)

	// sort the name by title for presentation
	sort.SliceStable(thingData.AttrNames, func(i int, j int) bool {
		k1 := thingData.AttrNames[i]
		prop1, _ := tdi.Properties[k1]
		k2 := thingData.AttrNames[j]
		prop2, _ := tdi.Properties[k2]
		return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
	})
	sort.SliceStable(thingData.ConfigNames, func(i int, j int) bool {
		k1 := thingData.ConfigNames[i]
		prop1, _ := tdi.Properties[k1]
		k2 := thingData.ConfigNames[j]
		prop2, _ := tdi.Properties[k2]
		return strings.ToLower(prop1.Title) < strings.ToLower(prop2.Title)
	})
	sort.SliceStable(thingData.EventNames, func(i int, j int) bool {
		k1 := thingData.EventNames[i]
		ev1, _ := tdi.Events[k1]
		k2 := thingData.EventNames[j]
		ev2, _ := tdi.Events[k2]
		return strings.ToLower(ev1.Title) < strings.ToLower(ev2.Title)
	})
	sort.SliceStable(thingData.ActionNames, func(i int, j int) bool {
		k1 := thingData.ActionNames[i]
		act1, _ := tdi.Actions[k1]
		k2 := thingData.ActionNames[j]
		act2, _ := tdi.Actions[k2]
		return strings.ToLower(act1.Title) < strings.ToLower(act2.Title)
	})

	thingData.DeviceType = ct.GetAtTypeTitle()

	// get the value of a make & model properties, if they exist
	// TODO: this is a bit of a pain to do. Is this a common problem?
	makeID, _ := tdi.GetPropertyOfVocabType(vocab.PropDeviceMake)
	if makeID != "" {
		makeValue := ct.GetPropertyOutput(makeID)
		if makeValue.Value.Text() != "" {
			thingData.MakeModel = makeValue.Value.Text() + ", "
		}
	}
	modelID, _ := tdi.GetPropertyOfVocabType(vocab.PropDeviceModel)
	if modelID != "" {
		modelValue := ct.GetPropertyOutput(modelID)
		if modelValue != nil && modelValue.Value.Text() != "" {
			thingData.MakeModel = thingData.MakeModel + modelValue.Value.Text()
		}
	}
	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderThingDetailsPageTemplate, thingData)
	sess.WritePage(w, buff, err)
}
