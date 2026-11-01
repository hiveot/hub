package directory

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/api/td"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
)

const RenderConfirmDeleteTDTemplate = "DeleteTD.gohtml"

type ConfirmDeleteTDTemplateData struct {
	ThingID            string
	TD                 *td.TD
	SubmitDeleteTDPath string
}

func RenderConfirmDeleteTD(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	var tdoc *td.TD

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	dirCl := sess.GetDirectory()
	tdoc, err = dirCl.RetrieveThing(thingID)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	tdParams := map[string]string{"thingID": thingID}
	data := ConfirmDeleteTDTemplateData{
		ThingID:            thingID,
		TD:                 tdoc,
		SubmitDeleteTDPath: utils.Substitute(src.DeleteThingPath, tdParams),
	}
	buff, err := app.RenderAppOrFragment(r, RenderConfirmDeleteTDTemplate, data)
	sess.WritePage(w, buff, err)
}

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
	dirCl := sess.GetDirectory()
	err = dirCl.DeleteThing(thingID)

	// reload the cached directory
	// cts := sess.GetConsumedThingsDirectory()
	// tdList, err = dirCl.RetrieveAllThings(0, 1000, true)

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
