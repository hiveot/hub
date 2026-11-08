package history

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"github.com/hiveot/hivekit/go/api/msg"
	"github.com/hiveot/hivekit/go/api/wot"
	"github.com/hiveot/hivekit/go/cells/consumer"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
)

// HistoryTemplateData holds the data for rendering a history table or graph
type HistoryTemplateData struct {
	ct consumer.ConsumedThing

	// history of this interaction output
	tdoc    *wot.TD
	affType msg.AffordanceType

	Name string

	// latest value
	Value msg.ThingValue
	// data type
	Schema *wot.DataSchema
	// unit symbol derived from the dataschema
	UnitSymbol string

	Title string // allow override to data description

	// property or event id as published
	//  {affordanceType}/{thingID}/{name}/
	//ID string //

	// history information
	EndTime     time.Time
	EndTimeStr  string
	DurationSec int
	// historical values used in template
	Values         []*msg.NotificationMessage
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
func (ht *HistoryTemplateData) AsJSON() string {
	dataList := []HistoryDataPoint{}

	for _, m := range ht.Values {
		yValue := m.Data
		if ht.Schema.Type == wot.DataTypeBool {
			boolValue := utils.DecodeAsBool(m.Data)
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
func (ht *HistoryTemplateData) GetObjectValues() []consumer.InteractionOutput {
	if ht.Schema.Type != "object" {
		iout := consumer.NewInteractionOutput(
			ht.tdoc, ht.affType, ht.Name, ht.Value.Raw, ht.Value.Timestamp)
		return []consumer.InteractionOutput{*iout}
	}

	objectValues := map[string]any{}
	objectAsJson, err := json.Marshal(ht.Value.Raw)
	if err == nil {
		err = json.Unmarshal(objectAsJson, &objectValues)
	}
	ioutList := make([]consumer.InteractionOutput, 0, len(objectValues))

	if err != nil {
		slog.Error("failed decoding object", "err", err.Error())
		return ioutList
	}
	// each object property is a row in the list
	// history of object values is currently not supported
	for name, schema := range ht.Schema.Properties {
		raw, found := objectValues[name]
		_ = found

		iout := consumer.NewInteractionOutput(
			ht.tdoc, ht.affType, ht.Name, raw, ht.Value.Timestamp)
		iout.Title = ht.Title + " - " + schema.Title
		ioutList = append(ioutList, *iout)
	}
	sort.Slice(ioutList, func(i, j int) bool {
		return ioutList[i].Title < ioutList[j].Title
	})
	return ioutList
}

// NextDay return the time +1 day
func (ht *HistoryTemplateData) NextDay() time.Time {
	return ht.EndTime.Add(time.Hour * 24)
}

// PrevDay return the time -1 day
func (ht *HistoryTemplateData) PrevDay() time.Time {
	return ht.EndTime.Add(-time.Hour * 24)
}

// CompareToday returns 0 if the timestamp is that of local time somewhere today
// this returns -1 if time is less than today and 1 if greater than today
//
// 'today' is different in that it refreshes if a value changes
func (ht *HistoryTemplateData) CompareToday() int {
	// 'today' accepts any time in the current local day
	yy, mm, dd := time.Now().Date()
	tsYY, tsmm, tsdd := ht.EndTime.Date()
	if yy == tsYY && mm == tsmm && dd == tsdd {
		return 0
	}
	diff := ht.EndTime.Compare(time.Now())
	return diff
}

// NewHistoryTemplateData reads the value history for the given time range
//
//	iout is the interaction output to display
//	values are the historical values to display
//	endTime of the end-time of the history range
//	duration to read (negative for history)
func NewHistoryTemplateData(
	// iout *consumer.InteractionOutput,
	tdoc *wot.TD,
	affType msg.AffordanceType,
	name string,
	values []*msg.NotificationMessage,
	endTime time.Time, duration time.Duration) (
	data *HistoryTemplateData, err error) {

	var title string
	// Get the current schema for the value to show
	var schema *wot.DataSchema

	switch affType {
	case msg.AffordanceTypeProperty:
		aff := tdoc.GetProperty(name)
		schema = &aff.DataSchema
		title = aff.Title
	case msg.AffordanceTypeEvent:
		aff := tdoc.GetEvent(name)
		schema = aff.Data
		title = aff.Title
	case msg.AffordanceTypeAction:
		aff := tdoc.GetAction(name)
		schema = aff.Output
		title = aff.Title
	default:
		err = fmt.Errorf("Unknown affordance '%s'", affType)
		return nil, err
	}

	histData := &HistoryTemplateData{
		// InteractionOutput: *iout,
		// ID:                iout.ID,
		Schema:  schema,
		tdoc:    tdoc,
		affType: affType,
		Name:    name,
		EndTime: endTime,
		// chart expects ISO timestamp: yyyy-mm-ddTHH:MM:SS.sss-07:00
		EndTimeStr:     utils.FormatUTCMilli(endTime),
		DurationSec:    int(duration.Seconds()),
		Stepped:        false,
		Values:         values,
		ItemsRemaining: false,
	}

	histData.UnitSymbol = schema.UnitSymbol()
	histData.Title = title //+ " of " + ct.Title
	//hs.DataSchema.Title = hs.Title
	histData.Stepped = schema.Type == wot.DataTypeBool

	// TODO: (if needed) if items remaining, get the rest in an additional call
	//hist := historypkg.NewReadHistoryClient(ct.GetConsumer())
	//hs.Values, hs.ItemsRemaining, err = hist.ReadHistory(
	//	iout.ThingID, iout.Name, timestamp, duration, 500)

	// Add the URL paths for navigating around the history
	pathParams := map[string]string{
		"affordanceType": string(affType),
		"thingID":        tdoc.ID,
		"name":           name,
	}
	prevDayTime := histData.PrevDay().Format(time.RFC3339)
	nextDayTime := histData.NextDay().Format(time.RFC3339)
	todayTime := time.Now().Format(time.RFC3339)
	histData.PrevDayPath = utils.Substitute(src.RenderHistoryTimePath+prevDayTime, pathParams)
	histData.NextDayPath = utils.Substitute(src.RenderHistoryTimePath+nextDayTime, pathParams)
	histData.TodayPath = utils.Substitute(src.RenderHistoryTimePath+todayTime, pathParams)
	histData.RenderHistoryLatestRow = utils.Substitute(src.RenderHistoryLatestValueRowPath, pathParams)
	histData.RenderThingDetailsPath = utils.Substitute(src.RenderThingDetailsPath, pathParams)
	return histData, err
}
