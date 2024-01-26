package directory

import (
	"encoding/json"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
	"net/http"
	"sort"
)

type DirGroup struct {
	Publisher string
	Things    []*things.TD
}

type DirectoryData struct {
	Groups map[string]*DirGroup
}

// Sort the given list of things and group them by publishing agent
// this returns a map of groups each containing an array of thing values
func sortByPublisher(tvList []things.ThingValue) *DirectoryData {
	dirData := &DirectoryData{
		Groups: make(map[string]*DirGroup),
	}

	// sort by agent+thingID for now
	sort.Slice(tvList, func(i, j int) bool {
		item1 := tvList[i]
		item2 := tvList[j]
		return item1.AgentID+item1.ThingID < item2.AgentID+item2.ThingID
	})
	for _, tv := range tvList {
		tplGroup, found := dirData.Groups[tv.SenderID]
		if !found {
			tplGroup = &DirGroup{
				Publisher: tv.SenderID,
				Things:    make([]*things.TD, 0),
			}
			dirData.Groups[tv.SenderID] = tplGroup
		}
		td := things.TD{}
		err := json.Unmarshal(tv.Data, &td)
		if err == nil {
			tplGroup.Things = append(tplGroup.Things, &td)
			if len(tplGroup.Things) == 0 {
				slog.Error("append failed")
			}
		}
	}
	return dirData
}

// RenderDirectory renders the list of Things
// This is invoked through a htmx call to fet the directory.
// The directory template is rendered on initial page load without any data, and
// again once the page is displayed in the browser, this time with data provided
// by this renderer.
func RenderDirectory(w http.ResponseWriter, r *http.Request) {
	var data = make(map[string]any)

	// 1: get session
	mySession, err := session.GetSession(w, r)
	if err == nil {
		hc := mySession.GetHubClient()
		rd := dirclient.NewReadDirectoryClient(hc)
		thingsList, err := rd.GetTDs(0, 100)
		if err == nil {
			dirGroups := sortByPublisher(thingsList)
			data["Directory"] = dirGroups
		} else {
			slog.Error("unable to load directory", "err", err)
		}
	}
	if err != nil {
		data["Error"] = err.Error()
	}
	data["PageNr"] = 1

	// don't cache the login
	views.TM.RenderTemplate(w, r, "directory.html", data)
}
