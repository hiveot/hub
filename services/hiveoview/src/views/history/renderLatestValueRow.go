package history

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/messaging/tputils"
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
// query param 'unit' with the unit to add to the row
func RenderLatestValueRow(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	name := chi.URLParam(r, "name")
	unit := r.URL.Query().Get("unit")
	fragment := ""

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	ct, err := sess.Consume(thingID)
	if err == nil {
		iout := ct.GetEventOutput(name)
		if iout == nil {
			iout = ct.GetPropertyOutput(name)
		}

		// FIXME: what if this is a property, not an event.
		// this will update the consumed thing value with an error
		// which is then displayed in properties elsewhere.
		//iout := ct.ReadEvent(name)
		// FIXME, if name is not an event then a 'not ane event' error
		// will be included in the values. These values are included in
		// the 'show value' even if it is a property.
		// SOLUTION: don't mix properties and events when displaying.
		//How to know this is a prop or event?

		fragment = fmt.Sprintf(addRowTemplate,
			tputils.DecodeAsDatetime(iout.Updated), iout.Value.Text(), unit)
	} else {
		fragment = fmt.Sprintf("")
	}
	_, _ = w.Write([]byte(fragment))
	return
}
