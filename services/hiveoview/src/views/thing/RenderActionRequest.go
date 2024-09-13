package thing

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/wot/consumedthing"
	"github.com/hiveot/hub/wot/tdd"
	"net/http"
	"time"
)

const RenderActionRequestTemplate = "RenderActionRequest.gohtml"

// ActionRequestTemplateData with data for the action request view
type ActionRequestTemplateData struct {
	// The thing and name the action belongs to
	ThingID string
	Name    string

	// the thing instance used to apply the action
	CT *consumedthing.ConsumedThing

	// Affordance of the action to issue containing the input dataschema
	Action *tdd.ActionAffordance

	// current state of the action
	// This uses the last received value related to this action.
	CurrentValue *consumedthing.InteractionOutput

	// the previous action request record
	LastActionRecord *digitwin.InboxRecord
	// previous action input (if any)
	LastActionInput consumedthing.DataSchemaValue
	// previous action timestamp (formatted)
	LastActionTime string
	// previous action age (formatted)
	LastActionAge string

	//
	SubmitActionRequestPath string
}

// Return the action affordance
func getActionAff(hc hubclient.IHubClient, thingID string, name string) (
	td *tdd.TD, actionAff *tdd.ActionAffordance, err error) {

	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
	if err != nil {
		return td, actionAff, err
	}
	err = json.Unmarshal([]byte(tdJson), &td)
	if err != nil {
		return td, actionAff, err
	}
	actionAff = td.GetAction(name)
	if actionAff == nil {
		return td, actionAff, fmt.Errorf("Action '%s' not found for Thing '%s'", name, thingID)
	}
	return td, actionAff, nil
}

// RenderActionRequest renders the action dialog.
// Path: /things/{thingID}/{name}
//
//	@param thingID this is the URL parameter
func RenderActionRequest(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	name := chi.URLParam(r, "name")
	var hc hubclient.IHubClient
	//var lastAction *digitwin.InboxRecord

	// Read the TD being displayed
	sess, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	ct, err := sess.Consume(thingID)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
	}

	_, actionAff, err := getActionAff(hc, thingID, name)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	data := ActionRequestTemplateData{
		ThingID: thingID,
		Name:    name,
		Action:  actionAff,
		CT:      ct,
	}
	cv, found := ct.GetValue(name)
	if found {
		data.CurrentValue = cv
	}

	// get last action request that was received
	// reading a latest value is optional
	lar, err := digitwin.InboxReadLatest(hc, name, thingID)
	if err == nil {
		data.LastActionRecord = &lar
		//data.PrevValue = &lastActionRecord
		updatedTime, _ := dateparse.ParseAny(data.LastActionRecord.Updated)
		data.LastActionTime = updatedTime.Format(time.RFC1123)
		data.LastActionAge = utils.Age(updatedTime)
		data.LastActionInput = consumedthing.NewDataSchemaValue(data.LastActionRecord.Input)
	}

	pathArgs := map[string]string{"thingID": data.ThingID, "name": data.Name}
	data.SubmitActionRequestPath = utils.Substitute(src.PostActionRequestPath, pathArgs)

	buff, err := app.RenderAppOrFragment(r, RenderActionRequestTemplate, data)
	sess.WritePage(w, buff, err)
}

// RenderActionProgress renders the action progress status component.
// TODO
//
//	@param thingID this is the URL parameter
func RenderActionProgress(w http.ResponseWriter, r *http.Request) {
	messageID := chi.URLParam(r, "messageID")
	//action := thing.ActionAffordance{}
	_ = messageID
}
