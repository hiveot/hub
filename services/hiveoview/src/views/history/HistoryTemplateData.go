package history

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/wot/td"
	"time"
)

// HistoryTemplateData holds the data for rendering a history table or graph
type HistoryTemplateData struct {
	AffordanceType string
	ThingID        string
	Title          string // allow override to data description
	Name           string
	DataSchema     td.DataSchema // dataschema of event/property key
	UnitSymbol     string        // unit of this data

	// history information
	Timestamp      time.Time
	TimestampStr   string
	DurationSec    int
	Values         []*messaging.ThingValue
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
		yValue := m.Output
		if ht.DataSchema.Type == vocab.WoTDataTypeBool {
			boolValue := tputils.DecodeAsBool(m.Output)
			yValue = 0
			if boolValue {
				yValue = 1
			}

		}
		dataList = append(dataList,
			HistoryDataPoint{X: m.Updated, Y: yValue})
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

// NewHistoryTemplateData reads the value history for the given time range
//
//	ct is the consumed thing to read the data from
//	affType affordance type
//	name of the event or property in the TD
//	timestamp of the end-time of the history range
//	duration to read (negative for history)
func NewHistoryTemplateData(ct *consumedthing.ConsumedThing,
	affType, name string, timestamp time.Time, duration time.Duration) (
	data *HistoryTemplateData, err error) {

	hs := HistoryTemplateData{
		AffordanceType: affType,
		ThingID:        ct.ThingID,
		Name:           name,
		Title:          ct.Title,
		Timestamp:      timestamp,
		// chart expects ISO timestamp: yyyy-mm-ddTHH:MM:SS.sss-07:00
		TimestampStr:   utils.FormatUTCMilli(timestamp),
		DurationSec:    int(duration.Seconds()),
		Stepped:        false,
		Values:         nil,
		ItemsRemaining: false,
	}
	// Get the current schema for the value to show
	iout := ct.GetValue(affType, name)

	hs.DataSchema = iout.Schema
	hs.UnitSymbol = iout.UnitSymbol()
	hs.Title = iout.Title + " of " + ct.Title
	hs.DataSchema.Title = hs.Title
	hs.Stepped = iout.Schema.Type == vocab.WoTDataTypeBool

	// TODO: (if needed) if items remaining, get the rest in an additional call
	//hs.Values, hs.ItemsRemaining, err = ct.ReadHistory(name, timestamp, duration)

	hist := historyclient.NewReadHistoryClient(ct.GetConsumer())
	hs.Values, hs.ItemsRemaining, err = hist.ReadHistory(
		ct.ThingID, name, timestamp, duration, 500)

	// Add the URL paths for navigating around the history
	pathParams := map[string]string{"affordanceType": affType, "thingID": ct.ThingID, "name": name}
	prevDayTime := hs.PrevDay().Format(time.RFC3339)
	nextDayTime := hs.NextDay().Format(time.RFC3339)
	todayTime := time.Now().Format(time.RFC3339)
	hs.PrevDayPath = tputils.Substitute(src.RenderHistoryTimePath+prevDayTime, pathParams)
	hs.NextDayPath = tputils.Substitute(src.RenderHistoryTimePath+nextDayTime, pathParams)
	hs.TodayPath = tputils.Substitute(src.RenderHistoryTimePath+todayTime, pathParams)
	hs.RenderHistoryLatestRow = tputils.Substitute(src.RenderHistoryLatestValueRowPath, pathParams)
	hs.RenderThingDetailsPath = tputils.Substitute(src.RenderThingDetailsPath, pathParams)
	return &hs, err
}
