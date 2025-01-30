package directory

import (
	"github.com/go-chi/chi/v5"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

const RenderConfirmDeleteTDTemplate = "RenderConfirmDeleteTD.gohtml"

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

	tdJson, err = digitwin.ThingDirectoryReadTD(sess.GetConsumer(), thingID)
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
		SubmitDeleteTDPath: tputils.Substitute(src.DeleteThingPath, tdParams),
	}
	buff, err := app.RenderAppOrFragment(r, RenderConfirmDeleteTDTemplate, data)
	sess.WritePage(w, buff, err)
}
