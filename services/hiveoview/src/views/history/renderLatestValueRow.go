package history

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"net/http"
)

// Add the latest event value to the history table and to the history chart
// This returns a html fragment with the table entry and some JS code to update chartjs.

const addRowTemplate = `
	<li>
 		<div>%s</div>
		<div>%v %s</div>
	</li>
`

// RenderLatestValueRow renders a single table row with the 'latest'
// thing event or property value.
//
// Intended to update the history table data on sse event.
//
// This is supposed to be temporary until events contain all message data
// and a JS function can format the table row, instead of calling the server.
//
// This is a stopgap for now.
//
// @param thingID to view
// @param key whose value to return
func RenderLatestValueRow(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	name := chi.URLParam(r, "name")

	// Read the TD being displayed and its latest values
	mySession, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		mySession.WriteError(w, err, 0)
		return
	}

	//latestValues, err := thing.GetLatest(thingID, hc)
	latestEvents, err := digitwin.OutboxReadLatest(
		hc, vocab.MessageTypeEvent, []string{name}, "", thingID)
	if err != nil {
		mySession.WriteError(w, err, 0)
		return
	}
	evmap, err := hubclient.NewThingMessageMapFromSource(latestEvents)
	if err == nil {
		tm := evmap[name]
		if tm != nil {
			// TODO: get unit symbol
			fragment := fmt.Sprintf(addRowTemplate,
				tm.GetUpdated("WT"), tm.Data, "")

			_, _ = w.Write([]byte(fragment))
			return
		}
		err = errors.New("cant find name: " + name)
	}
	mySession.WriteError(w, err, 0)
}