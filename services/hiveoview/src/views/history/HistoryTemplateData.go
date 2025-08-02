package history

import (
	"encoding/json"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"log/slog"
	"sort"
	"time"
)

// HistoryTemplateData holds the data for rendering a history table or graph
type HistoryTemplateData struct {
	// history of this interaction output
	consumedthing.InteractionOutput

	//AffordanceType messaging.AffordanceType
	Title string // allow override to data description

	// property or event id as published
	//  {affordanceType}/{thingID}/{name}/
	//ID string //

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
		yValue := m.Data
		if ht.Schema.Type == vocab.WoTDataTypeBool {
			boolValue := tputils.DecodeAsBool(m.Data)
			yValue = 0
			if boolValue {
				yValue = 1
			}

		}
		dataList = append(dataList,
			HistoryDataPoint{X: m.Timestamp, Y: yValue})
	}
	dataJSON, _ := json.Marshal(dataList)
	return string(dataJSON)
}

// GetObjectValues returns the history as an array of interaction objects
// If this is an object then return an element for each object property, otherwise
// return an array with 1 element.
func (ht HistoryTemplateData) GetObjectValues() []consumedthing.InteractionOutput {
	if ht.Schema.Type != "object" {
		return []consumedthing.InteractionOutput{ht.InteractionOutput}
	}

	objectValues := map[string]any{}
	objectAsJson, err := json.Marshal(ht.Value.Raw)
	if err == nil {
		err = json.Unmarshal(objectAsJson, &objectValues)
	}
	values := make([]consumedthing.InteractionOutput, 0, len(objectValues))

	if err != nil {
		slog.Error("failed decoding object", "err", err.Error())
		return values
	}
	// each object property is a row in the list
	// history of object values is currently not supported
	for name, schema := range ht.Schema.Properties {
		raw, found := objectValues[name]
		_ = found
		value := consumedthing.NewDataSchemaValue(raw)

		iout := ht.InteractionOutput // copy
		iout.Title = ht.Title + " - " + schema.Title
		iout.Name = name
		iout.Schema = schema
		iout.Value = value
		values = append(values, iout)
	}
	sort.Slice(values, func(i, j int) bool {
		return values[i].Title < values[j].Title
	})
	return values
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
//	iout is the initeraction output to display
//	values are the historical values to display
//	timestamp of the end-time of the history range
//	duration to read (negative for history)
func NewHistoryTemplateData(
	iout *consumedthing.InteractionOutput,
	values []*messaging.ThingValue,
	timestamp time.Time, duration time.Duration) (
	data *HistoryTemplateData, err error) {

	hs := HistoryTemplateData{
		InteractionOutput: *iout,
		//ID:                iout.ID,
		Timestamp: timestamp,
		// chart expects ISO timestamp: yyyy-mm-ddTHH:MM:SS.sss-07:00
		TimestampStr:   utils.FormatUTCMilli(timestamp),
		DurationSec:    int(duration.Seconds()),
		Stepped:        false,
		Values:         values,
		ItemsRemaining: false,
	}
	// Get the current schema for the value to show
	//iout := ct.GetValue(affType, name)

	//hs.DataSchema = iout.Schema
	//hs.UnitSymbol = iout.UnitSymbol()
	hs.Title = iout.Title //+ " of " + ct.Title
	//hs.DataSchema.Title = hs.Title
	hs.Stepped = iout.Schema.Type == vocab.WoTDataTypeBool

	// TODO: (if needed) if items remaining, get the rest in an additional call
	//hist := historyclient.NewReadHistoryClient(ct.GetConsumer())
	//hs.Values, hs.ItemsRemaining, err = hist.ReadHistory(
	//	iout.ThingID, iout.Name, timestamp, duration, 500)

	// Add the URL paths for navigating around the history
	pathParams := map[string]string{"affordanceType": string(iout.AffordanceType), "thingID": iout.ThingID, "name": iout.Name}
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
