package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"net/http"
)

// Write the raw TD
func RenderThingRaw(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	var tdJSON string
	var tdPretty []byte
	// Read the TD being displayed and its latest values
	sess, hc, err := session.GetSessionFromContext(r)
	if err == nil {
		tdJSON, err = digitwin.DirectoryReadDTD(hc, thingID)
	}
	if err == nil {
		// re-marshal with pretty-print JSON
		var tdObj any
		err = json.Unmarshal([]byte(tdJSON), &tdObj)
		tdPretty, _ = json.MarshalIndent(tdObj, "", "    ")
	}
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
	} else {
		w.Write(tdPretty)
		w.WriteHeader(http.StatusOK)
	}
}
