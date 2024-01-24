package directory

import (
	"encoding/json"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"github.com/hiveot/hub/core/directory/dirclient"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
	"net/http"
	"sort"
)

const SensorIcon = "import"
const ServiceIcon = "cube-outline"
const ActuatorIcon = "export"
const ControllerIcon = "usb" //"molecule"

// TODO: icons from config
var deviceTypeIcons = map[string]string{
	vocab.DeviceTypeBinarySwitch: ActuatorIcon,
	vocab.DeviceTypeBinding:      ServiceIcon,
	vocab.DeviceTypeCamera:       SensorIcon,
	vocab.DeviceTypeGateway:      ControllerIcon,
	vocab.DeviceTypeService:      ServiceIcon,
	vocab.DeviceTypeMultisensor:  SensorIcon,
	vocab.DeviceTypeSensor:       SensorIcon,
	vocab.DeviceTypeThermometer:  SensorIcon,
	// the following types should be changed in the binding to use the vocabulary
	"Binary Sensor":     SensorIcon,
	"Binary Switch":     ActuatorIcon,
	"Multilevel Sensor": SensorIcon,
	"Multilevel Switch": ActuatorIcon,
	"Static Controller": ControllerIcon,
	"error":             "alert-circle",
	"":                  "",
}

// TemplateThing is an intermediate struct for mapping ThingTD to the template IDs
type TemplateThing struct {
	Publisher string
	Icon      string
	ThingID   string
	Created   string
	Name      string
	Type      string
}

type TemplateGroup struct {
	Publisher string
	Things    []TemplateThing
}

// Sort the given list of things and group them by publishing agent
// this returns a map of groups each containing an array of thing values
func sortByPublisher(tvList []things.ThingValue) map[string]*TemplateGroup {
	groups := make(map[string]*TemplateGroup)

	// sort by agent+thingID for now
	sort.Slice(tvList, func(i, j int) bool {
		item1 := tvList[i]
		item2 := tvList[j]
		return item1.AgentID+item1.ThingID < item2.AgentID+item2.ThingID
	})
	for _, tv := range tvList {
		tplGroup, found := groups[tv.SenderID]
		if !found {
			tplGroup = &TemplateGroup{
				Publisher: tv.SenderID,
				Things:    make([]TemplateThing, 0),
			}
			groups[tv.SenderID] = tplGroup
		}
		td := things.TD{}
		err := json.Unmarshal(tv.Data, &td)
		if err == nil {
			icon := deviceTypeIcons[td.DeviceType]
			tplThing := TemplateThing{
				Publisher: tv.AgentID,
				Icon:      icon,
				ThingID:   tv.ThingID,
				Created:   td.Created,
				Name:      td.Title,
				Type:      td.DeviceType,
			}
			tplGroup.Things = append(tplGroup.Things, tplThing)
			if len(tplGroup.Things) == 0 {
				slog.Error("append failed")
			}
		}
	}
	return groups
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
			groupedThings := sortByPublisher(thingsList)
			data["Total"] = len(thingsList)
			data["Groups"] = groupedThings
		}
	}
	if err != nil {
		data["Error"] = err.Error()
	}
	data["PageNr"] = 1

	// don't cache the login
	views.TM.RenderTemplate(w, r, "directory.html", data)
}
