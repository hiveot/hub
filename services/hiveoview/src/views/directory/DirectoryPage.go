package directory

import (
	"bytes"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
)

const RenderDirectoryPageTemplate = "DirectoryPage.gohtml"
const DefaultMaxThings = 1000

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
	_, sess, err := session.GetSessionFromContext(r)
	if err != nil {
		sess.WriteError(w, err, 0)
		return
	}
	// cts := sess.GetConsumedThingsDirectory()
	dirCl := sess.GetDirectory()
	tdMap, cacheOnly := dirCl.RetrieveAllThings(0, DefaultMaxThings)
	if cacheOnly {
		err = fmt.Errorf("unable to load directory, using cached values")
		slog.Error(err.Error())
		sess.SendNotify(session.NotifyError, "", err.Error())
	}

	agentGroups := GroupByAgent(tdMap)
	data := DirectoryTemplateData{}
	data.Groups = agentGroups

	if err == nil {
		// full render or fragment render
		buff, err = app.RenderAppOrFragment(r, RenderDirectoryPageTemplate, data)
	}
	sess.WritePage(w, buff, err)
}
