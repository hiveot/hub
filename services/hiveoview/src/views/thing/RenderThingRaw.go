package thing

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	jsoniter "github.com/json-iterator/go"
	"net/http"
)

// Write the raw TD
func RenderThingRaw(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	var tdJSON string
	var tdPretty []byte
	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err == nil {
		tdJSON, err = digitwin.DirectoryReadTD(sess.GetHubClient(), thingID)
	}
	if err == nil {
		// re-marshal with pretty-print JSON
		var tdObj any
		err = jsoniter.UnmarshalFromString(tdJSON, &tdObj)
		tdPretty, _ = json.MarshalIndent(tdObj, "", "    ")
	}
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
	} else {
		w.Write(tdPretty)
		w.WriteHeader(http.StatusOK)
	}
}
