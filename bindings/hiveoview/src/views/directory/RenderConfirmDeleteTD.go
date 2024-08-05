package directory

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	thing "github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"net/http"
)

const RenderConfirmDeleteTDTemplate = "RenderConfirmDeleteTD.gohtml"
const SubmitDeleteTDPath = "/directory/{thingID}"

type ConfirmDeleteTDTemplateData struct {
	ThingID            string
	TD                 *thing.TD
	SubmitDeleteTDPath string
}

func RenderConfirmDeleteTD(w http.ResponseWriter, r *http.Request) {
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
	tdParams := map[string]string{"thingID": thingID}
	data := ConfirmDeleteTDTemplateData{
		ThingID:            thingID,
		TD:                 &td,
		SubmitDeleteTDPath: utils.Substitute(SubmitDeleteTDPath, tdParams),
	}
	app.RenderAppOrFragment(w, r, RenderConfirmDeleteTDTemplate, data)
}
