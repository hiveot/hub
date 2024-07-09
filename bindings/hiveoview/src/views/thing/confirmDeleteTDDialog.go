package thing

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	thing "github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
)

type ConfirmDeleteTDTemplateData struct {
	ThingID string
	TD      *thing.TD
}

func RenderConfirmDeleteTDDialog(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	td := thing.TD{}
	tdJson := ""

	// Read the TD being displayed and its latest values
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		mySession.WriteError(w, err, http.StatusBadRequest)
		return
	}

	tdJson, err = digitwin.DirectoryReadTD(hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJson), &td)
	}
	if err != nil {
		mySession.WriteError(w, err, http.StatusBadRequest)
		return
	}
	data := ConfirmDeleteTDTemplateData{
		ThingID: thingID,
		TD:      &td,
	}
	app.RenderAppOrFragment(w, r, "confirmDeleteTDDialog.gohtml", data)
}

// PostDeleteTD handles removal of a thing TD document
func PostDeleteTD(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	tdJSON := ""
	td := thing.TD{}
	var hc hubclient.IHubClient

	// get the hub client connection and read the existing TD
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		mySession.WriteError(w, err, http.StatusBadRequest)
		return
	}

	tdJSON, err = digitwin.DirectoryReadTD(hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJSON), &td)
	}

	// delete the TD
	if err == nil {
		slog.Info("Deleting TD", slog.String("thingID", thingID))
		err = digitwin.DirectoryRemoveTD(hc, thingID)
	}

	// report the result
	if err != nil {
		mySession.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	msgText := fmt.Sprintf("Thing '%s' successfully removed from the directory", td.Title)
	slog.Info(msgText, "thingID", td.ID)
	mySession.SendNotify(session.NotifySuccess, msgText)
	// navigate back to the directory.
	// http.Redirect doesn't work but using HX-Redirect header does.
	// see also: https://www.reddit.com/r/htmx/comments/188oqx5/htmx_form_submission_issue_redirecting_on_success/
	//http.Redirect(w, r, "/app/directory", http.StatusMovedPermanently)
	w.Header().Add("HX-Redirect", "/app/directory")
	w.WriteHeader(http.StatusOK)
}
