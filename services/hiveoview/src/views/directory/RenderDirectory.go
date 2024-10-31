package directory

import (
	"bytes"
	"fmt"
	session2 "github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"log/slog"
	"net/http"
)

const DirectoryTemplate = "RenderDirectory.gohtml"

//type DirGroup struct {
//	AgentID string
//	Things  []*things.TD
//}

type DirectoryGroup AgentThings

type DirectoryTemplateData struct {
	Groups []*AgentThings
	//PageNr      int
}

// RenderDirectory renders the directory of Things.
//
// This supports both a full and fragment rendering.
// Fragment rendering using htmx must use the #directory target.
// To view the directory, the #directory hash must be included at the end of the URL.
// E.g.: /directory/#directory
func RenderDirectory(w http.ResponseWriter, r *http.Request) {
	//var data = make(map[string]any)
	//var tdList []*things.TD
	var buff *bytes.Buffer

	// 1: get session
	_, sess, err := session2.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	cts := sess.GetConsumedThingsDirectory()

	tdMap, err := cts.ReadDirectory(false)
	if err != nil {
		err = fmt.Errorf("unable to load directory: %w", err)
		slog.Error(err.Error())
		sess.SendNotify(session2.NotifyError, err.Error())
	}

	agentGroups := GroupByAgent(tdMap)
	data := DirectoryTemplateData{}
	data.Groups = agentGroups

	if err == nil {
		// full render or fragment render
		buff, err = app.RenderAppOrFragment(r, DirectoryTemplate, data)
	}
	sess.WritePage(w, buff, err)
}
