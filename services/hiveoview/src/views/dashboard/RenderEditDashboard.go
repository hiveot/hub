package dashboard

import (
	"github.com/hiveot/hub/messaging/tputils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"github.com/teris-io/shortid"
	"html/template"
	"log/slog"
	"net/http"
)

const RenderEditDashboardTemplateFile = "RenderEditDashboard.gohtml"

type RenderEditDashboardTemplateData struct {
	Dashboard           session.DashboardModel
	SubmitEditDashboard string
	Background          template.URL // safe embedded image
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
		Dashboard:           newDashboard,
		SubmitEditDashboard: tputils.Substitute(src.PostDashboardEditPath, pathArgs),
	}
	data.Background = template.URL(data.Dashboard.Background)
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
	data := RenderEditDashboardTemplateData{
		Dashboard:           cdc.CurrentDashboard(),
		SubmitEditDashboard: getDashboardPath(src.PostDashboardEditPath, cdc),
	}
	data.Background = template.URL(data.Dashboard.Background)
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
	bgImg := r.PostFormValue("background")

	slog.Info("SubmitEditDashboard", "SenderID", cdc.clientID,
		"dashboardID", cdc.dashboardID)
	// add or update the dashboard
	dashboard := cdc.CurrentDashboard()
	dashboard.ID = cdc.dashboardID
	dashboard.Title = newTitle
	dashboard.Background = bgImg
	err = cdc.clientModel.UpdateDashboard(&dashboard)

	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusOK)

	// Notify the dashboard it has been updated so it will reload the fragment
	ev := getDashboardPath(src.DashboardUpdatedEvent, cdc)
	sess.SendSSE(ev, "")

}
