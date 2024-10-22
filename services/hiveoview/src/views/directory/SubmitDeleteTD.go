package directory

import (
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/services/hiveoview/src"
	session2 "github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/wot/tdd"
	"log/slog"
	"net/http"
)

// SubmitDeleteTD handles removal of a thing TD document
func SubmitDeleteTD(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	tdJSON := ""
	td := tdd.TD{}
	var hc hubclient.IConsumerClient

	// get the hub client connection and read the existing TD
	sess, hc, err := session2.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	tdJSON, err = digitwin.DirectoryReadDTD(hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJSON), &td)
	}

	// delete the TD
	if err == nil {
		slog.Info("Deleting TD", slog.String("thingID", thingID))
		err = digitwin.DirectoryRemoveDTD(hc, thingID)
	}
	cts := sess.GetConsumedThingsDirectory()
	// reload the cached directory
	cts.ReadDirectory(true)

	// report the result
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	msgText := fmt.Sprintf("Thing '%s' successfully removed from the directory", td.Title)
	slog.Info(msgText, "thingID", td.ID)
	sess.SendNotify(session2.NotifySuccess, msgText)
	// navigate back to the directory.
	// http.Redirect doesn't work but using HX-Redirect header does.
	// see also: https://www.reddit.com/r/htmx/comments/188oqx5/htmx_form_submission_issue_redirecting_on_success/
	//http.Redirect(w, r, "/app/directory", http.StatusMovedPermanently)
	w.Header().Add("HX-Redirect", src.RenderThingDirectoryPath)
	w.WriteHeader(http.StatusOK)
}
