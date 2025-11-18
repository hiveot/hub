package tile

import (
	"fmt"
	"html"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hivekit/go/messaging"
	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/services/hiveoview/src/session"
)

// RenderTileSourceRow renders a single table row with the tile 'source'
// description.
//
// This returns the html for the 'sources' table row in the edit tile view.
//
// FIXME: this is so horribly complicated for something so simple.
// All the information is already known in the RenderSelectSources view.
// Mentally you update the list of sources, not add data as HTML so this is
// a cognitive mismatch.
//
// Option 1: forget htmx and use JS between edit view and source selection.
// Handle it similar to a Select element where the selection is in a value
// property that is submitted.
//
//	pro: easier to use. Make it a form component in the edit view.
//	con: how?
//
// Option 2: create an edit object that is constantly updated during edit
//
//	pro: easier to work it into the regular htmx patter of post and get
//	con: editing is a client side activity, not server side???
//	con: ridiculous to have to do this for filling a simple input list
//
// Option 3: post custom html directly from the htmx dialog
//
//	pro: simplest
//	con: how to send html to target with htmx without a server roundtrip?
//
// Intended to update the list of sources in edit the tile dialog
// @param affType affordance type, "action", "event", "property"
// @param thingID of the thing the event belongs to
// @param key of the event affordance to add as source
func RenderTileSourceRow(w http.ResponseWriter, r *http.Request) {
	affType := chi.URLParam(r, "affordanceType")
	thingID := chi.URLParam(r, "thingID")
	name := chi.URLParam(r, "name")

	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	// obtain the TD and the event/action affordance to display
	var sourceID = fmt.Sprintf("%s/%s/%s", affType, thingID, name)

	cts := sess.GetConsumedThingsDirectory()
	// get the latest event values of this source
	var iout *consumedthing.InteractionOutput
	ct, err := cts.Consume(thingID)
	if ct != nil {
		iout = ct.GetValue(messaging.AffordanceType(affType), name)
	}
	if err == nil && iout == nil {
		err = fmt.Errorf("RenderTileSourceRow: No such affordance!?: %s", sourceID)
	}
	if err != nil {
		sess.WriteError(w, err, http.StatusInternalServerError)
		return
	}

	// if no value was ever received then use n/a
	latestValue := iout.Value.Text() + " " + iout.UnitSymbol()
	latestUpdated := utils.FormatAge(iout.Timestamp)
	title := ct.Title + " " + iout.Title

	// the input hidden hold the real source value
	// this must match the list in RenderEditTile.gohtml
	// FIXME: this is ridiculous htmx. Use JS to simplify it?.
	htmlToAdd := fmt.Sprintf(""+
		"<li id='new-source' draggable='true'>"+
		"  <div class='h-row-centered drag-handle'>"+
		"     <iconify-icon style='font-size:24px' icon='mdi:drag'></iconify-icon>"+
		"  </div>"+
		"  <input type='hidden' name='sources' value='%s'/>"+
		"  <input name='sourceTitles' value='%s' title='%s' style='margin:0'/>"+
		"  <div>%s</div>"+
		"  <div>%s</div>"+
		"  <button type='button' class='h-row-centered outline h-icon-button' style='border:none'"+
		"    onclick='deleteRow(this.parentNode)'>"+
		"		<iconify-icon icon='mdi:delete'></iconify-icon>"+
		"  </button>"+
		"</li>",
		html.EscapeString(sourceID),
		html.EscapeString(title),
		html.EscapeString(sourceID),
		html.EscapeString(latestValue),
		html.EscapeString(latestUpdated))

	_, _ = w.Write([]byte(htmlToAdd))
}
