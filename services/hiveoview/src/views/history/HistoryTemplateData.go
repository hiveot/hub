package history

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/wot/consumedthing"
	"github.com/hiveot/hub/wot/protocolclients"
	"github.com/hiveot/hub/wot/tdd"
	"strconv"
	"time"
)

// HistoryTemplateData holds the data for rendering a history table or graph
type HistoryTemplateData struct {
	TD         *tdd.TD
	ThingID    string
	Title      string // allow override to data description
	Name       string
	DataSchema tdd.DataSchema // dataschema of event/property key

	// history information
	Timestamp      time.Time
	TimestampStr   string
	DurationSec    int
	Values         []*transports.ThingMessage
	ItemsRemaining bool // for paging, if supported
	Stepped        bool // stepped graph

	// navigation paths
	RenderHistoryLatestRow string // table row
	PrevDayPath            string
	NextDayPath            string
	TodayPath              string
	RenderThingDetailsPath string
}

type HistoryDataPoint struct {
	X string `json:"x"`
	Y any    `json:"y"`
}

// AsJSON returns the values as a json string
// Booleans are converted to 0 and 1
func (ht HistoryTemplateData) AsJSON() string {
	dataList := []HistoryDataPoint{}

	for _, m := range ht.Values {
		yValue := m.Data
		if ht.DataSchema.Type == vocab.WoTDataTypeBool {
			boolValue, _ := strconv.ParseBool(m.DataAsText())
			yValue = 0
			if boolValue {
				yValue = 1
			}

		}
		dataList = append(dataList,
			HistoryDataPoint{X: m.Created, Y: yValue})
	}
	dataJSON, _ := json.Marshal(dataList)
	return string(dataJSON)
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

// NewHistoryTemplateData reads the event history for the given time range
//
//	ct is the consumed thing to read the data from
//	name of the event or property in the TD
//	timestamp of the end-time of the history range
//	duration to read (negative for history)
func NewHistoryTemplateData(
	ct *consumedthing.ConsumedThing, name string, timestamp time.Time, duration time.Duration) (
	data *HistoryTemplateData, err error) {

	td := ct.GetThingDescription()
	hs := HistoryTemplateData{
		TD:           td,
		ThingID:      td.ID,
		Name:         name,
		Title:        td.Title,
		Timestamp:    timestamp,
		TimestampStr: timestamp.Format(utils.RFC3339Milli),
		DurationSec:  int(duration.Seconds()),
		//DataSchema:     nil,  // see below
		Stepped:        false,
		Values:         nil,
		ItemsRemaining: false,
	}
	// Get the current schema for the value to show
	iout := ct.GetEventValue(name)
	if iout != nil {
		hs.DataSchema = iout.Schema
		hs.Title = iout.Title + " of " + td.Title
		hs.DataSchema.Title = hs.Title
		hs.Stepped = (iout.Schema.Type == vocab.WoTDataTypeBool)
	}

	// TODO: (if needed) if items remaining, get the rest in an additional call
	hs.Values, hs.ItemsRemaining, err = ct.ReadHistory(name, timestamp, duration)

	// Add the URL paths for navigating around the history
	pathParams := map[string]string{"thingID": td.ID, "name": name}
	prevDayTime := hs.PrevDay().Format(time.RFC3339)
	nextDayTime := hs.NextDay().Format(time.RFC3339)
	todayTime := time.Now().Format(time.RFC3339)
	hs.PrevDayPath = utils.Substitute(src.RenderHistoryTimePath+prevDayTime, pathParams)
	hs.NextDayPath = utils.Substitute(src.RenderHistoryTimePath+nextDayTime, pathParams)
	hs.TodayPath = utils.Substitute(src.RenderHistoryTimePath+todayTime, pathParams)
	hs.RenderHistoryLatestRow = utils.Substitute(src.RenderHistoryLatestValueRowPath, pathParams)
	hs.RenderThingDetailsPath = utils.Substitute(src.RenderThingDetailsPath, pathParams)
	return &hs, err
}
