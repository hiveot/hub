package thing

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hub/services/hiveoview/src/session"
)

// Write the raw TD
func RenderThingRaw(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	var tdoc *wot.TD
	var tdPretty []byte

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err == nil {
		dirCl := sess.GetDirectory()
		tdoc, err = dirCl.RetrieveThing(thingID)
	}
	if err == nil {
		// re-marshal with pretty-print JSON
		tdPretty, _ = json.MarshalIndent(tdoc, "", "    ")
	}
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
	} else {
		w.Write(tdPretty)
		w.WriteHeader(http.StatusOK)
	}
}
