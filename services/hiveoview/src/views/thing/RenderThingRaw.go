package thing

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	jsoniter "github.com/json-iterator/go"
)

// Write the raw TD
func RenderThingRaw(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	var tdJSON string
	var tdPretty []byte
	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	co := sess.GetConsumer()
	if err == nil {

		tdJSON, err = co.RetrieveThing(thingID)
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
