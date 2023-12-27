package login

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
	"net/http"
)

// RenderConnectStatus renders the presentation of the client connection to the Hub message bus.
func RenderConnectStatus(w http.ResponseWriter, r *http.Request) {
	var sessionInfo *session.ClientSession

	data := map[string]any{
		"conn_icon":   "link_off",     // or "link"
		"conn_status": "disconnected", // connected, disconnected, connection error
	}
	sessionID, _ := r.Cookie("session")
	sm := session.GetSessionManager()
	if sessionID != nil {
		sessionInfo, _ = sm.GetSession(sessionID.Value)
	}
	if sessionInfo == nil {
		data["conn_icon"] = "link_off"
		data["conn_status"] = "Not connected"
	} else {
		hc := sessionInfo.GetConnection()
		cStat := hc.GetStatus()
		if cStat.ConnectionStatus == transports.Connected {
			data["conn_icon"] = "link"
			data["conn_status"] = "Connected"
		} else if cStat.ConnectionStatus == transports.ConnectFailed {
			data["conn_icon"] = "link_off"
			data["conn_status"] = "Failed to connect: " + cStat.LastError
		} else if cStat.ConnectionStatus == transports.Connecting {
			data["conn_icon"] = "leak_add"
			data["conn_status"] = "Connecting ... " + cStat.LastError
		} else {
			data["conn_icon"] = "link_off"
			data["conn_status"] = "not connected: " + cStat.LastError
		}
	}
	t := assets.GetTemplate("connectStatus.html")
	err := t.Execute(w, data)
	if err != nil {
		slog.Error("Error rendering template", "err", err)
	}
}
