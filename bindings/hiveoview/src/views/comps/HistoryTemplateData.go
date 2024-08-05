package comps

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/history/historyclient"
	"strconv"
	"time"
)

const RenderHistoryPath = "/value/{thingID}/{key}/history?time="

// HistoryTemplateData holds the data for rendering a history table or graph
type HistoryTemplateData struct {
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

	// navigation paths
	PrevDayPath string
	NextDayPath string
	TodayPath   string
}

type HistoryDataTable struct {
	X string `json:"x"`
	Y any    `json:"y"`
}

// AsJSON returns the values as a json string
// Booleans are converted to 0 and 1
func (ht HistoryTemplateData) AsJSON() string {
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

// NewHistoryTemplateData reads the event or property history for the given time range
//
//	key is the key of the event or property in the TD
//	timestamp of the end-time of the history range
//	duration nr of seconds to read (negative for history)
func NewHistoryTemplateData(hc hubclient.IHubClient,
	td *things.TD, key string, timestamp time.Time, duration int) (*HistoryTemplateData, error) {

	var err error
	hs := HistoryTemplateData{
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

	// Add the URL paths for navigating around the history
	pathParams := map[string]string{"thingID": td.ID, "key": key}
	prevDayTime := hs.PrevDay().Format(time.RFC3339)
	nextDayTime := hs.NextDay().Format(time.RFC3339)
	todayTime := time.Now().Format(time.RFC3339)
	hs.PrevDayPath = utils.Substitute(RenderHistoryPath+prevDayTime, pathParams)
	hs.NextDayPath = utils.Substitute(RenderHistoryPath+nextDayTime, pathParams)
	hs.TodayPath = utils.Substitute(RenderHistoryPath+todayTime, pathParams)

	return &hs, err
}
