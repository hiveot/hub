package directory

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

const RenderConfirmDeleteTDTemplate = "RenderConfirmDeleteTD.gohtml"

type ConfirmDeleteTDTemplateData struct {
	ThingID            string
	TD                 *tdd.TD
	SubmitDeleteTDPath string
}

func RenderConfirmDeleteTD(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	td := tdd.TD{}
	tdJson := ""

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	tdJson, err = digitwin.DirectoryReadTD(sess.GetHubClient(), thingID)
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
