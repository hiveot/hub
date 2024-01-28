package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"net/http"
)

// GetConnectStatusProps populates the data map with the mqtt/nats connection status
// intended for rendering the connection status.
func GetConnectStatusProps(data map[string]any, r *http.Request) {
	cs, _ := session.GetSession(nil, r)
	connIcon := "link-off"
	connText := "Disconnected"
	isConnected := false
	if cs == nil {
		connIcon = "link-off"
		connText = "Session not established"
	} else {
		cStat := cs.GetStatus()
		if cStat.LastError != nil {
			connText = cStat.LastError.Error()
		}
		if cStat.ConnectionStatus == transports.Connected {
			connIcon = "link"
			connText = "Connected to the Hub"
			isConnected = true
		} else if cStat.ConnectionStatus == transports.ConnectFailed {
			connIcon = "link-off"
			connText = "Connection failed (" + connText + ")"
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
	if isConnected {
		data["isConnected"] = isConnected
	}
}

// RenderConnectStatus renders the presentation of the client connection to the Hub message bus.
// This only renders the fragment. On a full page refresh this renders inside the base.html
func RenderConnectStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	GetConnectStatusProps(data, r)

	views.TM.RenderTemplate(w, r, "connectStatus.html", data)
}
