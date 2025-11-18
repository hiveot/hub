package directory

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	digitwin "github.com/hiveot/hivehub/runtime/digitwin/api"
	"github.com/hiveot/hivehub/services/hiveoview/src"
	"github.com/hiveot/hivehub/services/hiveoview/src/session"
	"github.com/hiveot/hivehub/services/hiveoview/src/views/app"
	"github.com/hiveot/hivekitgo/utils"
	"github.com/hiveot/hivekitgo/wot/td"
	jsoniter "github.com/json-iterator/go"
)

const RenderConfirmDeleteTDTemplate = "DeleteTD.gohtml"

type ConfirmDeleteTDTemplateData struct {
	ThingID            string
	TD                 *td.TD
	SubmitDeleteTDPath string
}

func RenderConfirmDeleteTD(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	td := td.TD{}
	tdJson := ""

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	tdJson, err = digitwin.ThingDirectoryRetrieveThing(sess.GetConsumer(), thingID)
	if err == nil {
		err = jsoniter.UnmarshalFromString(tdJson, &td)
	}
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	tdParams := map[string]string{"thingID": thingID}
	data := ConfirmDeleteTDTemplateData{
		ThingID:            thingID,
		TD:                 &td,
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
	err = digitwin.ThingDirectoryDeleteThing(sess.GetConsumer(), thingID)

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
