package directory

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	"net/http"
)

const RenderConfirmDeleteTDTemplate = "RenderConfirmDeleteTD.gohtml"
const SubmitDeleteTDPath = "/directory/{thingID}"

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
	sess, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		// TODO: redirect to login?
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}

	tdJson, err = digitwin.DirectoryReadTD(hc, thingID)
	if err == nil {
		err = json.Unmarshal([]byte(tdJson), &td)
	}
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	tdParams := map[string]string{"thingID": thingID}
	data := ConfirmDeleteTDTemplateData{
		ThingID:            thingID,
		TD:                 &td,
		SubmitDeleteTDPath: utils.Substitute(SubmitDeleteTDPath, tdParams),
	}
	buff, err := app.RenderAppOrFragment(r, RenderConfirmDeleteTDTemplate, data)
	sess.WritePage(w, buff, err)
}
