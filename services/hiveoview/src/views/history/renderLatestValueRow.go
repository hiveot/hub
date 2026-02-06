package history

import (
	"fmt"
	"html"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/lib/messaging"
	"github.com/hiveot/hub/services/hiveoview/src/session"
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
//	@param affType event, property, action
//	@param thingID to view
//	@param key whose value to return
//	query param 'unit' with the unit to add to the row
func RenderLatestValueRow(w http.ResponseWriter, r *http.Request) {
	affType := chi.URLParam(r, "affordanceType")
	thingID := chi.URLParam(r, "thingID")
	name := chi.URLParam(r, "name")
	unit := html.EscapeString(r.URL.Query().Get("unit"))
	fragment := ""

	// Read the TD being displayed and its latest values
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	ct, err := sess.Consume(thingID)
	if err == nil {
		iout := ct.GetValue(messaging.AffordanceType(affType), name)
		fragment = fmt.Sprintf(addRowTemplate,
			utils.FormatDateTime(iout.Timestamp, "S"), iout.Value.Text(), unit)
	} else {
		fragment = fmt.Sprintf("")
	}
	_, _ = w.Write([]byte(fragment))
	return
}
