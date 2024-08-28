package thing

import (
	"encoding/json"
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"net/http"
	"time"
)

const RenderActionRequestTemplate = "RenderActionRequest.gohtml"
const SubmitActionRequestPath = "/action/{thingID}/{key}"

// ActionRequestTemplateData with data for the action request view
type ActionRequestTemplateData struct {
	// The thing the action belongs to
	ThingID string
	TD      *tdd.TD
	// key of action
	Key string
	// the action being viewed in case of an action
	Action *tdd.ActionAffordance
	Input  *SchemaValue
	Output *SchemaValue
	// the message with the action
	Msg hubclient.ThingMessage
	// current delivery status
	Status hubclient.DeliveryStatus
	// Previous input value
	PrevValue *digitwin.InboxRecord
	// timestamp the action was last performed
	LastUpdated string
	// duration the action was last performed
	LastUpdatedAge string

	//
	SubmitActionRequestPath string
}

// Return the action affordance
func getActionAff(hc hubclient.IHubClient, thingID string, key string) (
	td *tdd.TD, actionAff *tdd.ActionAffordance, err error) {

	tdJson, err := digitwin.DirectoryReadTD(hc, thingID)
	if err != nil {
		return td, actionAff, err
	}
	err = json.Unmarshal([]byte(tdJson), &td)
	if err != nil {
		return td, actionAff, err
	}
	actionAff = td.GetAction(key)
	if actionAff == nil {
		return td, actionAff, fmt.Errorf("Action '%s' not found for Thing '%s'", key, thingID)
	}
	return td, actionAff, nil
}

// RenderActionRequest renders the action dialog.
// Path: /things/{thingID/{key}
//
//	@param thingID this is the URL parameter
func RenderActionRequest(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")
	var hc hubclient.IHubClient
	//var lastAction *digitwin.InboxRecord

	data := ActionRequestTemplateData{
		ThingID: thingID,
		Key:     key,
	}
	// Read the TD being displayed
	sess, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	if err == nil {
		data.TD, data.Action, err = getActionAff(hc, thingID, key)
	}
	// last action that was submitted
	if err == nil {
		// reading a latest value is optional
		lastActionRecord, err2 := digitwin.InboxReadLatest(hc, key, thingID)
		if err2 == nil {
			data.PrevValue = &lastActionRecord
			updatedTime, _ := dateparse.ParseAny(lastActionRecord.Updated)
			data.LastUpdated = updatedTime.Format(time.RFC1123)
			data.LastUpdatedAge = utils.Age(updatedTime)
		}
	}
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	prevInputValue := ""
	if data.PrevValue != nil && data.PrevValue.Input != nil {
		prevInputValue = fmt.Sprintf("%v", data.PrevValue.Input)
	}
	if data.Action.Input != nil {
		data.Input = &SchemaValue{
			ThingID:    thingID,
			Key:        key,
			DataSchema: data.Action.Input,
			Value:      prevInputValue,
		}
	}
	if data.Action.Output != nil {
		data.Output = &SchemaValue{
			ThingID:    thingID,
			Key:        key,
			DataSchema: data.Action.Output,
			Value:      "",
		}
	}
	pathArgs := map[string]string{"thingID": data.ThingID, "key": data.Key}
	data.SubmitActionRequestPath = utils.Substitute(SubmitActionRequestPath, pathArgs)

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
