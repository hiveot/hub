package thing

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/wot/consumedthing"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/http"
)

const RenderEditPropertyTemplate = "RenderEditProperty.gohtml"

// Edit Thing Property data
type RenderEditPropertyTemplateData struct {
	ThingID    string
	Name       string
	DataSchema *tdd.DataSchema
	Value      string
	// The last known value of the property to edit
	PropertyValue      consumedthing.InteractionOutput
	SubmitPropertyPath string
}

func getPropAff(hc hubclient.IHubClient, thingID string, name string) (
	td *tdd.TD, propAff *tdd.PropertyAffordance, err error) {

	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
	if err != nil {
		return td, propAff, err
	}
	err = json.Unmarshal([]byte(tdJson), &td)
	if err != nil {
		return td, propAff, err
	}
	propAff = td.GetProperty(name)
	if propAff == nil {
		return td, propAff, fmt.Errorf("property '%s' not found for Thing '%s'", name, thingID)
	}
	return td, propAff, nil
}

// obtain a schema value instance from a thingID and key
// this pulls the TD from the server (todo: consider using a local cache)
func getConfigValue(
	hc hubclient.IHubClient, thingID string, name string) (sv RenderEditPropertyTemplateData, err error) {
	var valueStr string
	var dataValue any

	td, propAff, err := getPropAff(hc, thingID, name)
	_ = td
	if err != nil {
		return sv, err
	}

	names := []string{name}
	senderID := ""
	updated := ""
	propValues, err := digitwin.OutboxReadLatest(hc, "", names, "", thingID)
	if err == nil {
		// convert the property value to string for presentation
		// TODO: make this simpler
		tmRaw, found := propValues[name]
		if found {
			tm := hubclient.ThingMessage{}
			tmJson, _ := json.Marshal(tmRaw)
			_ = json.Unmarshal(tmJson, &tm)
			dataValue = tm.Data
			valueStr = fmt.Sprintf("%v", tm.Data)
			senderID = tm.SenderID
			updated = tm.GetUpdated()
		}
	}
	sv = RenderEditPropertyTemplateData{
		//TD:         &td,
		ThingID:    thingID,
		Name:       name,
		DataSchema: &propAff.DataSchema,
		Value:      valueStr,
		PropertyValue: consumedthing.InteractionOutput{
			Name:     name,
			Title:    propAff.Title,
			Schema:   propAff.DataSchema,
			Value:    consumedthing.NewDataSchemaValue(dataValue),
			Updated:  updated,
			SenderID: senderID,
		},
	}
	return sv, nil
}

// RenderEditProperty renders the view for Thing property configuration
// This sets the data properties for AgentID, ThingID, Name and Config
func RenderEditProperty(w http.ResponseWriter, r *http.Request) {
	var data RenderEditPropertyTemplateData

	thingID := chi.URLParam(r, "thingID")
	propName := chi.URLParam(r, "name")

	sess, hc, err := session.GetSessionFromContext(r)
	if err == nil {
		data, err = getConfigValue(hc, thingID, propName)
	}
	pathParams := map[string]string{"thingID": thingID, "name": propName}
	data.SubmitPropertyPath = utils.Substitute(src.PostThingPropertyEditPath, pathParams)
	if err != nil {
		slog.Error("RenderEditConfigDialog:", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	buff, err := app.RenderAppOrFragment(r, RenderEditPropertyTemplate, data)
	sess.WritePage(w, buff, err)
}
