package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/views/layouts"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"html/template"
	"log/slog"
	"net/http"
)

// GetApp renders the main application view
func GetApp(t *template.Template) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

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
			"Title":      "HiveOT",
			"theme":      "dark",
			"theme_icon": "dark_mode",
			"pages":      []string{"page1", "page2"},
			"connected":  isConnected,
		}
		layouts.RenderWithLayout(w, t, "app.html", "main.html", data)
	}
}
