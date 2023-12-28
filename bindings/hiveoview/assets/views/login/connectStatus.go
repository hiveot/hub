package login

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
	"net/http"
)

func GetConnectStatusProps(data map[string]any, r *http.Request) {
	cs := session.GetClientSession(r)
	connIcon := "link-off"
	connText := "Disconnected"
	if cs == nil {
		connIcon = "link-off"
		connText = "Session not yet established"
	} else {
		cStat := cs.GetStatus()
		if cStat.ConnectionStatus == transports.Connected {
			connIcon = "link"
			connText = "Connected as " + cs.LoginID
		} else if cStat.ConnectionStatus == transports.ConnectFailed {
			connIcon = "link-off"
			connText = "Failed to connect: " + cStat.LastError
		} else if cStat.ConnectionStatus == transports.Connecting {
			connIcon = "leak-add"
			connText = "Connecting ... " + cStat.LastError
		} else {
			connIcon = "link-off"
			connText = "Not connected; " + cStat.LastError
		}
	}
	data["conn_icon"] = connIcon
	data["conn_status"] = connText
}

// RenderConnectStatus renders the presentation of the client connection to the Hub message bus.
func RenderConnectStatus(w http.ResponseWriter, r *http.Request) {

	data := map[string]any{}
	GetConnectStatusProps(data, r)

	t := assets.GetTemplate("connectStatus.html")
	err := t.Execute(w, data)
	if err != nil {
		slog.Error("Error rendering template", "err", err)
	}
}
