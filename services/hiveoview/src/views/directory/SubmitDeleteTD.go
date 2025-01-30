package directory

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/wot/td"
	"log/slog"
	"net/http"
)

// SubmitDeleteTD handles removal of a thing TD document
func SubmitDeleteTD(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	tdi := td.TD{}

	// get the hub client connection and read the existing TD
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	slog.Info("Deleting TD", slog.String("thingID", thingID))
	err = digitwin.ThingDirectoryRemoveTD(sess.GetConsumer(), thingID)

	// reload the cached directory
	cts := sess.GetConsumedThingsDirectory()
	_, err = cts.ReadDirectory(true)

	// report the result
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	msgText := fmt.Sprintf("Thing '%s' successfully removed from the directory", tdi.Title)
	slog.Info(msgText, "thingID", tdi.ID)
	sess.SendNotify(session.NotifySuccess, "", msgText)
	// navigate back to the directory.
	// http.Redirect doesn't work but using HX-Redirect header does.
	// see also: https://www.reddit.com/r/htmx/comments/188oqx5/htmx_form_submission_issue_redirecting_on_success/
	//http.Redirect(w, r, "/app/directory", http.StatusMovedPermanently)
	w.Header().Add("HX-Redirect", src.RenderThingDirectoryPath)
	w.WriteHeader(http.StatusOK)
}
