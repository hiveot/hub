package history

import (
	"net/http"
	"strconv"
	"time"

	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
)

const RenderHistoryTemplate = "HistoryPage.gohtml"

// RenderHistoryPage renders a table with historical event values
// URL parameters:
// @param affType event, property or action affordance to view
// @param thingID to view
// @param name of event whose value to show
// @param timestamp the start or end time of the viewing period
// @param duration number of seconds to view. default is -24 hours
func RenderHistoryPage(w http.ResponseWriter, r *http.Request) {
	affType := chi.URLParam(r, "affordanceType")
	thingID := chi.URLParam(r, "thingID")
	name := chi.URLParam(r, "name")
	timestampStr := r.URL.Query().Get("time")
	durationStr := r.URL.Query().Get("duration")
	durationSec, _ := strconv.ParseInt(durationStr, 10, 32)
	if timestampStr == "" {
		// this formats as local time with zulu timezone!
		timestampStr = utils.FormatNowUTCMilli()
	}
	timestamp, err := dateparse.ParseAny(timestampStr)
	timestamp = timestamp.Local()
	if durationSec == 0 {
		durationSec = -24 * 3600
	}

	// key events are escaped as SSE doesn't allow spaces and slashes

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	cts := sess.GetConsumedThingsDirectory()
	// read the TD
	ct, err := cts.Consume(thingID)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	duration := time.Second * time.Duration(durationSec)
	iout := ct.GetValue(messaging.AffordanceType(affType), name)
	hist := historyclient.NewReadHistoryClient(ct.GetConsumer())
	values, itemsRemaining, err := hist.ReadHistory(
		iout.ThingID, iout.Name, timestamp, duration, 500)
	_ = itemsRemaining

	var data *HistoryTemplateData
	if err == nil {
		data, err = NewHistoryTemplateData(iout, values, timestamp, duration)
	}
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	// full render or fragment render
	buff, err := app.RenderAppOrFragment(r, RenderHistoryTemplate, data)
	sess.WritePage(w, buff, err)
}
