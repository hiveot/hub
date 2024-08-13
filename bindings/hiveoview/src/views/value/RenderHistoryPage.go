package value

import (
	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/comps"
	"github.com/hiveot/hub/lib/utils"
	"net/http"
	"strconv"
	"time"
)

const RenderHistoryTemplate = "RenderHistoryPage.gohtml"

// RenderHistoryPage renders a table with historical values
// URL parameters:
// @param thingID to view
// @param key whose value to show
// @param timestamp the start or end time of the viewing period
// @param duration number of seconds to view. default is -24 hours
func RenderHistoryPage(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")
	timestampStr := r.URL.Query().Get("time")
	durationStr := r.URL.Query().Get("duration")
	durationSec, _ := strconv.ParseInt(durationStr, 10, 32)
	if timestampStr == "" {
		timestampStr = time.Now().Format(utils.RFC3339Milli)
	}
	timestamp, err := dateparse.ParseAny(timestampStr)
	if durationSec == 0 {
		durationSec = -24 * 3600
	}

	// key events are escaped as SSE doesn't allow spaces and slashes

	// Read the TD being displayed and its latest values
	sess, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	// read the TD
	td, err := session.ReadTD(hc, thingID)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	data, err := comps.NewHistoryTemplateData(
		hc, td, key, timestamp, int(durationSec))

	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderHistoryTemplate, data)
	sess.WritePage(w, buff, err)
}
