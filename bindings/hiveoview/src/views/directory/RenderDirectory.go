package directory

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views/app"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
	"sort"
)

const DirectoryTemplate = "RenderDirectory.gohtml"

type DirGroup struct {
	AgentID string
	Things  []*things.TD
}

type DirectoryTemplateData struct {
	Groups map[string]*DirGroup
	PageNr int
}

// Group the thing list by agent
// this returns a map of groups each containing an array of thing values
func groupThings(tdList []*things.TD) map[string]*DirGroup {
	dirData := make(map[string]*DirGroup)

	// group Things by their agent. The agent is the thing prefix
	for _, td := range tdList {
		agentID, _ := things.SplitDigiTwinThingID(td.ID)
		tplGroup, found := dirData[agentID]
		if !found {
			tplGroup = &DirGroup{
				AgentID: agentID,
				Things:  make([]*things.TD, 0),
			}
			dirData[agentID] = tplGroup
		}
		tplGroup.Things = append(tplGroup.Things, td)
	}
	return dirData
}

// Sort the things in each group
func sortGroupThings(dirGroups map[string]*DirGroup) {
	for _, grp := range dirGroups {
		sort.Slice(grp.Things, func(i, j int) bool {
			tdI := grp.Things[i]
			tdJ := grp.Things[j]
			return tdI.Title < tdJ.Title
		})
	}
}

// RenderDirectory renders the directory of Things.
//
// This supports both a full and fragment rendering.
// Fragment rendering using htmx must use the #directory target.
// To view the directory, the #directory hash must be included at the end of the URL.
// E.g.: /directory/#directory
func RenderDirectory(w http.ResponseWriter, r *http.Request) {
	//var data = make(map[string]any)
	var tdList []*things.TD
	var buff *bytes.Buffer

	// 1: get session
	sess, hc, err := session.GetSessionFromContext(r)
	if err != nil {
		slog.Info("failed getting session. Redirecting to login", "err", err.Error())
		// assume this is an auth issue or expired session
		//mySession.Close()
		//http.Error(w, err.Error(), http.StatusUnauthorized)

		// FIXME: logout doesn't update URL to /login (need navigateto?)
		session.SessionLogout(w, r)
		return
	}

	//thingsList := make([]things.TD, 0)
	thingsList, err2 := digitwin.DirectoryReadTDs(hc, 300, 0)
	for _, tdJson := range thingsList {
		td := things.TD{}
		err = json.Unmarshal([]byte(tdJson), &td)
		if err == nil {
			tdList = append(tdList, &td)
		}
	}
	//resp, err2 := directory.ReadTDs(hc, directory.ReadTDsArgs{Limit: 200})
	data := DirectoryTemplateData{}
	err = err2
	if err == nil {
		dirGroups := groupThings(tdList)
		sortGroupThings(dirGroups)
		data.Groups = dirGroups
	} else {
		// the 'Directory' attribute is used by html know if to reload
		err = fmt.Errorf("unable to load directory: %w", err)
		slog.Error(err.Error())
	}

	if err == nil {
		data.PageNr = 1

		// full render or fragment render
		buff, err = app.RenderAppOrFragment(r, DirectoryTemplate, data)
	}
	sess.WritePage(w, buff, err)
}
