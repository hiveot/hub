package status

import (
	"bytes"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	app2 "github.com/hiveot/hub/services/hiveoview/src/views/app"
	"net/http"
)

const TemplateFile = "status.gohtml"

// RenderStatus renders the client status page
func RenderStatus(w http.ResponseWriter, r *http.Request) {
	var buff *bytes.Buffer
	sess, _, err := session.GetSessionFromContext(r)

	status := app2.GetConnectStatus(r)

	data := map[string]any{}
	data["Status"] = status

	// full render or fragment render
	if err == nil {
		buff, err = app2.RenderAppOrFragment(r, TemplateFile, data)
	}
	sess.WritePage(w, buff, err)
}
