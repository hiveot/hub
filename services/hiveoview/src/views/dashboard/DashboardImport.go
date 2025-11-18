package dashboard

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/hiveot/hivekit/go/utils"
	"github.com/hiveot/hub/services/hiveoview/src"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	jsoniter "github.com/json-iterator/go"
)

const NewDashboardFieldName = "new-dashboard"

type DashboardImportTemplateData struct {
	SubmitDashboardImportPath string
	NewDashboardFieldName     string
}

// RenderDashboardExport returns the raw dashboard JSON
func RenderDashboardExport(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := getDashboardContext(r, false)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	dash, _ := cdc.SelectedDashboard()
	dashJSON, _ := jsoniter.MarshalIndent(dash, "", "  ")
	w.Write(dashJSON)
}

// RenderDashboardImport renders a dialog to import an exported dashboard configuration
func RenderDashboardImport(w http.ResponseWriter, r *http.Request) {
	sess, cdc, err := getDashboardContext(r, false)
	_ = cdc
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	// full render or fragment render

	data := DashboardImportTemplateData{
		SubmitDashboardImportPath: getDashboardPath(src.PostDashboardImportPath, cdc),
		NewDashboardFieldName:     NewDashboardFieldName,
	}
	buff, err := app.RenderAppOrFragment(r, RenderDashboardImportTemplate, data)
	sess.WritePage(w, buff, err)
}

// SubmitDashboardImport replaces the current dashboard with the given one
func SubmitDashboardImport(w http.ResponseWriter, r *http.Request) {

	sess, cdc, err := getDashboardContext(r, true)
	if err != nil {
		sess.WriteError(w, err, http.StatusBadRequest)
		return
	}
	dashboard, _ := cdc.SelectedDashboard()
	newDashboardJson := r.PostFormValue(NewDashboardFieldName)

	if len(newDashboardJson) == 0 {
		slog.Warn("SubmitDashboardImport: missing dashboard content",
			"clientID", sess.GetClientID(),
			"dashboardID", dashboard.ID)
		sess.WriteError(w, errors.New("missing dashboard content"), http.StatusBadRequest)
		return
	}
	newDashboard := session.DashboardModel{}
	err = jsoniter.Unmarshal([]byte(newDashboardJson), &newDashboard)
	if err != nil {
		slog.Warn("SubmitDashboardImport: new dashboard content not json",
			"clientID", sess.GetClientID(),
			"dashboardID", dashboard.ID)
		sess.WriteError(w, errors.New("new dashboard content not valid json"), http.StatusBadRequest)
		return
	}

	slog.Info("SubmitDashboardImport", "SenderID", cdc.clientID,
		"dashboardID", cdc.dashboardID)

	sess.SendNotify(session.NotifySuccess, "",
		fmt.Sprintf("Dashboard '%s' was successfully imported", dashboard.Title))
	// replace the existing dashboard
	newDashboard.ID = dashboard.ID
	err = cdc.clientModel.UpdateDashboard(&newDashboard)
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
