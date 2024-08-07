package status

import (
	"bytes"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"net/http"
)

const TemplateFile = "status.gohtml"

// RenderStatus renders the client status page
func RenderStatus(w http.ResponseWriter, r *http.Request) {
	var buff *bytes.Buffer
	sess, _, err := session.GetSessionFromContext(r)

	status := app.GetConnectStatus(r)

	data := map[string]any{}
	data["Status"] = status

	// full render or fragment render
	if err == nil {
		buff, err = app.RenderAppOrFragment(r, TemplateFile, data)
	}
	sess.WritePage(w, buff, err)
}
