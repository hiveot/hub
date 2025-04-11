package tile

import (
	"github.com/hiveot/hub/lib/consumedthing"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"strings"
)

const EditTileTemplate = "RenderEditTile.gohtml"

type EditTileTemplateData struct {
	Dashboard session.DashboardModel
	Tile      session.DashboardTile
	// Values of the tile sources by sourceID (affType/thingID/name)
	// FIXME: used consumed thing directory to get values?
	ctDir *consumedthing.ConsumedThingsDirectory
	//Values map[string]*consumedthing.InteractionOutput
	// human labels for each tile type
	TileTypeLabels map[string]string

	// navigation paths
	RenderSelectTileSourcesPath string // dialog for tile sources selector
	SubmitEditTilePath          string // submit the edited tile
}

func (data EditTileTemplateData) GetTypeLabel(typeID string) string {
	label, found := session.TileTypesLabels[typeID]
	if !found {
		return typeID
	}
	return label
}

// GetValue returns the value of a tile source
func (data EditTileTemplateData) GetValue(tileSource session.TileSource) string {
	ct, err := data.ctDir.Consume(tileSource.ThingID)
	if err != nil {
		// should never happen, but just in case
		return err.Error()
	}
	iout := ct.GetValue(tileSource.AffordanceType, tileSource.Name)
	if iout != nil {
		unitSymbol := iout.UnitSymbol()
		return iout.Value.Text() + " " + unitSymbol
	}
	return ""
}
func (data EditTileTemplateData) GetUpdated(tileSource session.TileSource) string {
	ct, err := data.ctDir.Consume(tileSource.ThingID)
	if err != nil {
		return ""
	}
	iout := ct.GetValue(tileSource.AffordanceType, tileSource.Name)
	if iout != nil {
		utils.FormatDateTime(iout.Updated)
	}
	return ""
}

// RenderEditTile renders the Tile editor dialog
// If the tile does not exist a new tile will be created
func RenderEditTile(w http.ResponseWriter, r *http.Request) {
	sess, ctc, err := GetTileContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// assign a 'new tile' ID if needed
	if ctc.tileID == "" {
		ctc.tileID = shortid.MustGenerate()
	}
	ctDir := sess.GetConsumedThingsDirectory()
	data := EditTileTemplateData{
		Dashboard:                   ctc.dashboard,
		Tile:                        ctc.tile,
		TileTypeLabels:              session.TileTypesLabels,
		RenderSelectTileSourcesPath: getTilePath(src.RenderTileSelectSourcesPath, ctc),
		SubmitEditTilePath:          getTilePath(src.PostTileEditPath, ctc),
		ctDir:                       ctDir,
	}
	buff, err := app.RenderAppOrFragment(r, EditTileTemplate, data)
	sess.WritePage(w, buff, err)
}

// SubmitEditTile updated or creates a tile and adds it to the dashboard
// This expects an input form with the following fields:
//   - title
//   - tileType
//   - sources (an array of affType/thingID/name strings)
//   - sourceTitles an array of corresponding titles of the sources
func SubmitEditTile(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := GetTileContext(r, false)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	err = r.ParseForm()

	slog.Info("SubmitEditTile",
		slog.String("clientID", cdc.clientID),
		slog.String("dashboardID", cdc.dashboardID),
		slog.String("tileID", cdc.tileID),
	)
	// The edit tile form has a list of sources for the thingID/key and
	// a list of titles of each source. This is the only way I knew on how
	// to pass lists in Forms.
	newTitle := r.FormValue("title")
	tileType := r.FormValue("tileType")
	bgTransparency := r.FormValue("bgTransparency")
	bgColor := r.FormValue("bgColor")
	sources, _ := r.Form["sources"]
	sourceTitles, _ := r.Form["sourceTitles"]

	tile, found := cdc.dashboard.GetTile(cdc.tileID)
	if !found {
		// this is a new tile
		tile = cdc.dashboard.NewTile(cdc.tileID, "", session.TileTypeCard)
	}
	tile.Title = newTitle
	tile.TileType = tileType
	tile.BackgroundTransparency = bgTransparency
	tile.BackgroundColor = bgColor
	tile.Sources = make([]session.TileSource, 0)

	// Convert the list of sources from the form to a TileSource object.
	if sources != nil {
		// each source consists of affType/thingID/name
		i := 0
		for _, s := range sources {
			sourceTitle := "?"
			if i < len(sourceTitles) {
				sourceTitle = sourceTitles[i]
			}
			parts := strings.Split(s, "/")
			if len(parts) >= 3 {
				tileSource := session.TileSource{
					AffordanceType: parts[0],
					ThingID:        parts[1],
					Name:           parts[2],
					Title:          sourceTitle,
				}
				tile.Sources = append(tile.Sources, tileSource)
			}
			i++
		}
	}

	// add the new tile to the dashboard
	cdc.dashboard.Tiles[tile.ID] = tile
	cdc.clientModel.UpdateDashboard(&cdc.dashboard)

	if found {
		// Notify the UI that the tile has changed. The eventName was provided
		// in RenderTile.
		eventName := strings.ReplaceAll(src.TileUpdatedEvent, "{tileID}", tile.ID)
		sess.SendSSE(eventName, "")
	} else {
		// this is a new tile. Notify the dashboard
		eventName := strings.ReplaceAll(src.DashboardUpdatedEvent, "{dashboardID}", cdc.dashboardID)
		sess.SendSSE(eventName, "")
	}
	sess.WriteError(w, err, http.StatusOK)
}
