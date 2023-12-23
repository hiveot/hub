package login

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"html/template"
	"log/slog"
	"net/http"
)

// GetConnectStatus renders the presentation of the client connection to the Hub message bus.
func GetConnectStatus(templates *template.Template, sm *session.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sessionInfo *session.SessionInfo

		data := map[string]any{
			"conn_icon":   "link_off",     // or "link"
			"conn_status": "disconnected", // connected, disconnected, connection error
		}
		sessionID, _ := r.Cookie("SessionID")
		if sessionID != nil {
			sessionInfo, _ = sm.GetSession(sessionID.Value)
		}
		if sessionInfo != nil && sessionInfo.HC != nil {
			cStat, cInfo := sessionInfo.HC.ConnectionStatus()
			if cStat == transports.Connected {
				data["conn_icon"] = "link"
				data["conn_status"] = "Connected (" + cInfo + ")"
			} else if cStat == transports.ConnectFailed {
				data["conn_icon"] = "link_off"
				data["conn_status"] = "Failed to connect: " + cInfo
			} else if cStat == transports.Connecting {
				data["conn_icon"] = "leak_add"
				data["conn_status"] = "Connecting ... " + cInfo
			} else {
				data["conn_status"] = "unknown (" + cInfo + ")"
			}
		}

		err := templates.ExecuteTemplate(w, "connectStatus.html", data)
		if err != nil {
			slog.Error("Error rendering template", "err", err)
		}
	}
}
