package history

import (
	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
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
	name := chi.URLParam(r, "name")
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
	sess, _, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	cts := sess.GetConsumedThingsSession()
	// read the TD
	ct, err := cts.Consume(thingID)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	duration := time.Second * time.Duration(durationSec)
	data, err := NewHistoryTemplateData(ct, name, timestamp, duration)

	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderHistoryTemplate, data)
	sess.WritePage(w, buff, err)
}