package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"github.com/hiveot/hub/bindings/hiveoview/assets/components"
	"github.com/hiveot/hub/bindings/hiveoview/assets/templates/layouts"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"html/template"
	"log/slog"
	"net/http"
)

// GetApp renders the main application view
func GetApp(t *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// workaround for error: 'cannot clone layout.html after it has executed'
		// this error makes no sense. layouthtml is always cloned before execution
		t2, err := assets.ParseTemplates()
		if err != nil {
			return
		}

		var isConnected bool = false
		//c.String(http.StatusOK, "the home page")
		siContext := r.Context().Value("session")
		if siContext != nil {
			si := siContext.(*session.SessionInfo)
			slog.Info("found session", "loginID", si.LoginID)
			connStat, _ := si.HC.ConnectionStatus()
			isConnected = connStat == transports.Connected
		}
		data := map[string]any{
			//"Title":      "HiveOT",
			"theme":      "dark",
			"theme_icon": "bi-sun", // bi-sun bi-moon-fill
			//"pages":      []string{"page1", "page2"},
			"connected": isConnected,
		}
		components.SetAppbarProps(data, "HiveOT", "/static/logo.svg", []string{"page1", "page2"})
		layouts.RenderWithLayout(w, t2, "app.html", "", data)
	}
}
