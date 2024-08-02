package comps

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/history/historyclient"
	"net/http"
	"strconv"
	"time"
)

// Add the latest event value to the history table and to the history chart
// This returns a html fragment with the table entry and some JS code to update chartjs.

const addRowTemplate = `
	<li>
 		<div>%s</div>
		<div>%v %s</div>
	</li>
`

// HistorySourceData holds the data for rendering a history table or graph
type HistorySourceData struct {
	TD         *things.TD
	ThingID    string
	Title      string // allow override to data description
	Key        string
	DataSchema things.DataSchema // dataschema of event/property key

	// history information
	Timestamp      time.Time
	TimestampStr   string
	DurationSec    int
	Values         []*things.ThingMessage
	ItemsRemaining bool // for paging, if supported
}

type HistoryDataTable struct {
	X string `json:"x"`
	Y any    `json:"y"`
}

// AsJSON returns the values as a json string
// Booleans are converted to 0 and 1
func (ht HistorySourceData) AsJSON() string {
	dataList := []HistoryDataTable{}

	for _, m := range ht.Values {
		yValue := m.Data
		if ht.DataSchema.Type == vocab.WoTDataTypeBool {
			boolValue, _ := strconv.ParseBool(m.DataAsText())
			yValue = 0
			if boolValue {
				yValue = 1
			}

		}
		//dataList = append(dataList,
		//	HistoryDataTable{X: m.Created, Y: m.DataAsText()})
		dataList = append(dataList,
			HistoryDataTable{X: m.Created, Y: yValue})
	}
	dataJSON, _ := json.Marshal(dataList)
	return string(dataJSON)
}

// NextDay return the time +1 day
func (ht HistorySourceData) NextDay() time.Time {
	return ht.Timestamp.Add(time.Hour * 24)
}

// PrevDay return the time -1 day
func (ht HistorySourceData) PrevDay() time.Time {
	return ht.Timestamp.Add(-time.Hour * 24)
}

// CompareToday returns 0 if the timestamp is that of local time somewhere today
// this returns -1 if time is less than today and 1 if greater than today
//
// 'today' is different in that it refreshes if a value changes
func (ht HistorySourceData) CompareToday() int {
	// 'today' accepts any time in the current local day
	yy, mm, dd := time.Now().Date()
	tsYY, tsmm, tsdd := ht.Timestamp.Date()
	if yy == tsYY && mm == tsmm && dd == tsdd {
		return 0
	}
	diff := ht.Timestamp.Compare(time.Now())
	return diff
}

// NewHistorySourceData reads the event or property history for the given time range
//
//	key is the key of the event or property in the TD
//	timestamp of the end-time of the history range
//	duration nr of seconds to read (negative for history)
func NewHistorySourceData(hc hubclient.IHubClient,
	td *things.TD, key string, timestamp time.Time, duration int) (*HistorySourceData, error) {

	var err error
	hs := HistorySourceData{
		TD:           td,
		ThingID:      td.ID,
		Key:          key,
		Title:        td.Title,
		Timestamp:    timestamp,
		TimestampStr: timestamp.Format(utils.RFC3339Milli),
		DurationSec:  duration,
		//DataSchema:     nil,  // see below
		Values:         nil,
		ItemsRemaining: false,
	}
	evAff := td.GetEvent(key)
	if evAff != nil {
		hs.DataSchema = *evAff.Data
		hs.Title = td.Title + ", " + evAff.Title
		hs.DataSchema.Title = evAff.Title
	} else {
		propAff := td.GetProperty(key)
		if propAff != nil {
			hs.DataSchema = propAff.DataSchema
			hs.Title = td.Title + ", " + propAff.Title
			hs.DataSchema.Title = propAff.Title
		}
	}

	limit := 1000
	hs.Values = make([]*things.ThingMessage, 0)

	hist := historyclient.NewReadHistoryClient(hc)
	hs.Values, hs.ItemsRemaining, err = hist.ReadHistory(td.ID, key, timestamp, duration, limit)
	return &hs, err
}

// RenderHistoryLatest renders a single table row with the 'latest' value.
// Intended to update the history table data.
//
// This is supposed to be temporary until events contain all message data
// and a JS function can format the table row, instead of calling the server.
//
// This is a stopgap for now.
//
// @param thingID to view
// @param key whose value to return
func RenderHistoryLatest(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")

	// Read the TD being displayed and its latest values
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		mySession.WriteError(w, err, 0)
		return
	}

	//latestValues, err := thing.GetLatest(thingID, hc)
	latestEvents, err := digitwin.OutboxReadLatest(
		hc, []string{key}, vocab.MessageTypeEvent, "", thingID)
	if err != nil {
		mySession.WriteError(w, err, 0)
		return
	}
	evmap, err := things.NewThingMessageMapFromSource(latestEvents)
	if err == nil {
		tm := evmap[key]
		if tm != nil {
			// TODO: get unit symbol
			fragment := fmt.Sprintf(addRowTemplate,
				tm.GetUpdated("WT"), tm.Data, "")

			_, _ = w.Write([]byte(fragment))
			return
		}
		err = errors.New("cant find key: " + key)
	}
	mySession.WriteError(w, err, 0)
}
