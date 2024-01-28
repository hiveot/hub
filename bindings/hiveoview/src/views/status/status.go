package status

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

type AppStatus struct {
	LoginID       string
	ConnectStatus string
	Error         string
}

// RenderStatus renders the client status page
func RenderStatus(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	s, err := session.GetSession(w, r)
	if err != nil {
		data["Status"] = &AppStatus{Error: err.Error()}
	} else {
		data["Status"] = &AppStatus{
			LoginID:       s.GetHubClient().ClientID(),
			ConnectStatus: string(s.GetStatus().ConnectionStatus),
		}
	}

	// full render or fragment render
	app.RenderAppOrFragment(w, r, "status.html", data)
}
