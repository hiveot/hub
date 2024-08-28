package directory

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/http"
)

// redirect path after deleting the TD
const RenderDirectoryPath = "/directory"

// SubmitDeleteTD handles removal of a thing TD document
func SubmitDeleteTD(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	tdJSON := ""
	td := tdd.TD{}
	var hc hubclient.IHubClient

	// get the hub client connection and read the existing TD
	sess, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		sess.WriteError(w, err, http.StatusBadRequest)
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
	cts := sess.GetConsumedThingsSession()
	// reload the cached directory
	cts.ReadDirectory(true)

	// report the result
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	msgText := fmt.Sprintf("Thing '%s' successfully removed from the directory", td.Title)
	slog.Info(msgText, "thingID", td.ID)
	sess.SendNotify(session.NotifySuccess, msgText)
	// navigate back to the directory.
	// http.Redirect doesn't work but using HX-Redirect header does.
	// see also: https://www.reddit.com/r/htmx/comments/188oqx5/htmx_form_submission_issue_redirecting_on_success/
	//http.Redirect(w, r, "/app/directory", http.StatusMovedPermanently)
	w.Header().Add("HX-Redirect", RenderDirectoryPath)
	w.WriteHeader(http.StatusOK)
}
