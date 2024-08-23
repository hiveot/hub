package tile

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/wot/tdd"
	"net/http"
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
// @param thingID of the thing the event belongs to
// @param key of the event affordance to add as source
func RenderTileSourceRow(w http.ResponseWriter, r *http.Request) {
	thingID := chi.URLParam(r, "thingID")
	key := chi.URLParam(r, "key")

	// just format the data
	sess, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	cts := sess.GetConsumedThingsSession()
	var aff *tdd.EventAffordance
	td := cts.GetTD(thingID)
	if td != nil {
		aff = td.GetEvent(key)
	}
	if aff == nil {
		err = fmt.Errorf("thingID '%s' or event '%s' not found",
			thingID, key)
		sess.WriteError(w, err, 0)
		return
	}
	latestEvents, err := digitwin.OutboxReadLatest(
		hc, []string{key}, vocab.MessageTypeEvent, "", thingID)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	evmap, err := hubclient.NewThingMessageMapFromSource(latestEvents)
	if err == nil {
		unitSymbol := aff.Data.UnitSymbol()
		tm := evmap[key]
		sourceRef := thingID + "/" + key
		title := td.Title + ": " + aff.Title
		latestValue := "n/a"
		latestUpdated := "n/a"
		// if no value was ever received then use n/a
		if tm != nil {
			latestValue = tm.DataAsText() + " " + unitSymbol
			latestUpdated = tm.GetUpdated()
		}
		// the input hidden hold the real source value
		// this must match the list in RenderEditTile.gohtml
		// FIXME: this is ridiculous htmx. Use JS to simplify it.
		htmlToAdd := fmt.Sprintf(""+
			"<li>"+
			"  <input type='hidden' name='sources' value='%s'/>"+
			"  <button type='button' class='h-row outline h-icon-button'"+
			"    onclick='deleteRow(this.parentNode)'>"+
			"		<iconify-icon icon='mdi:delete'></iconify-icon>"+
			"	</button>"+
			"  <input name='sourceTitles' value='%s' title='%s'/>"+
			"  <div>%s</div>"+
			"  <div>%s</div>"+
			"</li>",
			sourceRef, title, sourceRef, latestValue, latestUpdated)

		_, _ = w.Write([]byte(htmlToAdd))
		return
	}
	sess.WriteError(w, err, 0)
}
