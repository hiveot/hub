package dashboard

import (
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/hiveot/hivehub/services/hiveoview/src"
	"github.com/hiveot/hivehub/services/hiveoview/src/session"
	"github.com/hiveot/hivehub/services/hiveoview/src/views/app"
	"github.com/hiveot/hivekitgo/utils"
	"github.com/teris-io/shortid"
)

const RenderEditDashboardTemplateFile = "EditDashboard.gohtml"

type RenderEditDashboardTemplateData struct {
	Dashboard          session.DashboardModel
	SubmitDashboard    string
	BackgroundImageURL template.URL // safe embedded image
}

// RenderAddDashboard renders the dialog for adding a dashboard
func RenderAddDashboard(w http.ResponseWriter, r *http.Request) {
	//sess, cdc, err := getDashboardContext(r, true)
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	newDashboard := session.NewDashboard(shortid.MustGenerate(), "New Dashboard")
	pathArgs := map[string]string{"dashboardID": newDashboard.ID}
	data := RenderEditDashboardTemplateData{
		Dashboard:       newDashboard,
		SubmitDashboard: utils.Substitute(src.PostDashboardEditPath, pathArgs),
	}
	data.BackgroundImageURL = template.URL(data.Dashboard.BackgroundImage)
	buff, err := app.RenderAppOrFragment(r, RenderEditDashboardTemplateFile, data)
	sess.WritePage(w, buff, err)
}

// RenderEditDashboard renders the dialog for configuring a dashboard
func RenderEditDashboard(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	dashboard, _ := cdc.SelectedDashboard()
	data := RenderEditDashboardTemplateData{
		Dashboard:       dashboard,
		SubmitDashboard: getDashboardPath(src.PostDashboardEditPath, cdc),
	}
	// source-URL overrides any existing image
	if data.Dashboard.BackgroundURL != "" {
		data.Dashboard.BackgroundImage = data.Dashboard.BackgroundURL
	}
	data.BackgroundImageURL = template.URL(data.Dashboard.BackgroundImage)
	buff, err := app.RenderAppOrFragment(r, RenderEditDashboardTemplateFile, data)
	sess.WritePage(w, buff, err)
}

// SubmitEditDashboard updates the dashboard configuration
func SubmitEditDashboard(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// 'title' is the form field from gohtml
	newTitle := r.PostFormValue("title")
	bgURL := r.PostFormValue("backgroundURL")
	bgEnabled := r.PostFormValue("backgroundEnabled") == "on" // "on" or ""
	bgImage := r.PostFormValue("backgroundImage")
	bgInterval := r.PostFormValue("reloadInterval")
	bgIntervalInt, _ := strconv.ParseInt(bgInterval, 10, 32)
	locked := r.PostFormValue("locked") == "on"    // "on" or ""
	floatTiles := r.PostFormValue("float") == "on" // "on" or ""

	slog.Info("SubmitDashboard", "SenderID", cdc.clientID,
		"dashboardID", cdc.dashboardID)

	// add or update the dashboard; bgURL takes precedence
	if bgURL != "" {
		bgImage = bgURL
	}
	dashboard, _ := cdc.SelectedDashboard()
	dashboard.ID = cdc.dashboardID // the URL determines the ID
	dashboard.Title = newTitle
	dashboard.BackgroundEnabled = bgEnabled
	dashboard.BackgroundImage = bgImage
	dashboard.BackgroundURL = bgURL
	dashboard.BackgroundReloadInterval = int(bgIntervalInt)
	dashboard.Locked = locked
	dashboard.Grid.Float = floatTiles

	err = cdc.clientModel.UpdateDashboard(&dashboard)

	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	// do a full reload in case title and dashboard selection changed
	args := map[string]string{URLParamDashboardID: cdc.dashboardID}
	dashboardPath := utils.Substitute(src.RenderDashboardPath, args)
	w.Header().Add("HX-Redirect", dashboardPath)
	w.WriteHeader(http.StatusOK)
}
