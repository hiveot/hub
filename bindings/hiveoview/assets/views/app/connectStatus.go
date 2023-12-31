package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"net/http"
)

func GetConnectStatusProps(data map[string]any, r *http.Request) {
	cs, _ := session.GetSession(nil, r)
	connIcon := "link-off"
	connText := "Disconnected"
	if cs == nil {
		connIcon = "link-off"
		connText = "Session not yet established"
	} else {
		cStat := cs.GetStatus()
		if cStat.LastError != nil {
			connText = cStat.LastError.Error()
		}
		if cStat.ConnectionStatus == transports.Connected {
			connIcon = "link"
			connText = "Connected to Hub"
		} else if cStat.ConnectionStatus == transports.ConnectFailed {
			connIcon = "link-off"
			connText = "Connect failed (" + connText + ")"
		} else if cStat.ConnectionStatus == transports.Connecting {
			connIcon = "leak-off"
			connText = "Reconnecting (" + connText + ")"
		} else {
			connIcon = "link-off"
			connText = connText
		}
	}
	data["conn_icon"] = connIcon
	data["conn_status"] = connText
}

// RenderConnectStatus renders the presentation of the client connection to the Hub message bus.
func RenderConnectStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	GetConnectStatusProps(data, r)
	assets.RenderMain(w, r, "connectStatus.html", data)
}
