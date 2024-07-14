package history

import (
	"fmt"
	"github.com/araddon/dateparse"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/history/historyclient"
	"net/http"
	"strconv"
	"time"
)

const HistoryPageTemplate = "historyPage.gohtml"

type HistoryTemplateData struct {
	ThingID      string
	Key          string
	TimestampStr string
	DurationSec  int
	//
	Timestamp      time.Time
	TD             *things.TD
	Aff            *things.EventAffordance
	Values         []*things.ThingMessage
	ItemsRemaining bool
}

// NextDay return the time +1 day
func (ht HistoryTemplateData) NextDay() time.Time {
	return ht.Timestamp.Add(time.Hour * 24)
}

// PrevDay return the time -1 day
func (ht HistoryTemplateData) PrevDay() time.Time {
	return ht.Timestamp.Add(-time.Hour * 24)
}

// CompareToday returns 0 if the timestamp is that of local time somewhere today
// this returns -1 if time is less than today and 1 if greater than today
//
// 'today' is different in that it refreshes if a value changes
func (ht HistoryTemplateData) CompareToday() int {
	// 'today' accepts any time in the current local day
	yy, mm, dd := time.Now().Date()
	tsYY, tsmm, tsdd := ht.Timestamp.Date()
	if yy == tsYY && mm == tsmm && dd == tsdd {
		return 0
	}
	diff := ht.Timestamp.Compare(time.Now())
	return diff
}

// ReadHistoryData reads the history of a thing event
// This returns a list of event messages in 'latest-first sort order'
//
//	timestamp is the start time to search at
//	durationSec is positive to search forward and negative to search backwards
func ReadHistoryData(hc hubclient.IHubClient, thingID string, key string,
	timestamp time.Time, durationSec int) (items []*things.ThingMessage, itemsRemaining bool, err error) {

	limit := 1000
	items = make([]*things.ThingMessage, 0)
	var batch []*things.ThingMessage

	hist := historyclient.NewReadHistoryClient(hc)
	batch, itemsRemaining, err = hist.ReadHistory(thingID, key, timestamp, durationSec, limit)
	items = append(items, batch...)
	return items, itemsRemaining, nil
}

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

	data := HistoryTemplateData{
		ThingID:      thingID,
		Key:          key,
		TimestampStr: timestampStr,
		Timestamp:    timestamp,
		DurationSec:  int(durationSec),
	}

	// Read the TD being displayed and its latest values
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		mySession.WriteError(w, err, 0)
		return
	}
	// read the TD
	td, err := mySession.ReadTD(thingID)
	if err != nil {
		mySession.WriteError(w, err, 0)
		return
	}
	data.TD = td
	data.Aff = td.GetEvent(key)
	if data.Aff == nil {
		err = fmt.Errorf("event '%s' does not exist in Thing '%s'", key, thingID)
		mySession.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// read the history
	data.Values, data.ItemsRemaining, err = ReadHistoryData(
		hc, thingID, key, timestamp, int(durationSec))
	if err != nil {
		mySession.WriteError(w, err, 0)
		return
	}

	// full render or fragment render
	app.RenderAppOrFragment(w, r, HistoryPageTemplate, data)
}
