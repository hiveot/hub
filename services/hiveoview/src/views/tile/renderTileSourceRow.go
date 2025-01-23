package tile

import (
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/transports/tputils"
	"github.com/hiveot/hub/wot/consumedthing"
	"github.com/hiveot/hub/wot/td"
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
	name := chi.URLParam(r, "name")

	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	// obtain the TD and the event/action affordance to display
	var sourceRef = thingID + "/" + name

	cts := sess.GetConsumedThingsDirectory()
	tdi := cts.GetTD(thingID)
	if tdi == nil {
		// use an empty TD otherwise it won't be possible to delete the bad entry
		tdi = &td.TD{ID: thingID, Title: "Unknown Thing"}
		//sess.WriteError(w, errors.New("Unknown Thing id="+thingID), 0)
		//return
	}
	// get the latest event values of this source
	// FIXME: why is this needed?
	tv, err := digitwin.ValuesReadEvent(sess.GetHubClient(), name, thingID)
	if tv.Name == "" {
		tv, err = digitwin.ValuesReadProperty(sess.GetHubClient(), name, thingID)
	}
	io := consumedthing.NewInteractionOutputFromValue(&tv, tdi)
	if io == nil || tv.Name == "" {
		io = &consumedthing.InteractionOutput{
			ThingID: thingID,
			Name:    name,
			Title:   "Unknown affordance",
			Schema:  td.DataSchema{},
			Value:   consumedthing.DataSchemaValue{},
			Err:     errors.New("unknown affordance"),
		}
		//sess.WriteError(w, errors.New("Unknown affordance: "+name), 0)
		//return
	}
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}

	// if no value was ever received then use n/a
	latestValue := io.Value.Text() + " " + io.UnitSymbol()
	latestUpdated := tputils.DecodeAsDatetime(io.Updated)
	title := tdi.Title + " " + io.Title

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
		"  <input name='sourceTitles' value='%s' title='%s' style='margin:0'/>"+
		"  <div>%s</div>"+
		"  <div>%s</div>"+
		"</li>",
		sourceRef, title, sourceRef, latestValue, latestUpdated)

	_, _ = w.Write([]byte(htmlToAdd))
	return
}
