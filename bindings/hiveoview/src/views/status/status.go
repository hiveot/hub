package status

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

// RenderStatus renders the client status page
func RenderStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	s, err := session.GetSession(w, r)
	if err == nil {
		data["LoginID"] = s.GetHubClient().ClientID()
		data["ConnectStatus"] = s.GetStatus().ConnectionStatus
	}

	// just get the status data
	// TODO on first load the status page should fetch its content
	// so nothing to do here??
	app.RenderAppPages(w, r, data)
}
