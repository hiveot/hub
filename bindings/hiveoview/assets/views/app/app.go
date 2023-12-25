package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"github.com/hiveot/hub/bindings/hiveoview/assets/components"
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
	"net/http"
	"strings"
)

const templateName = "app.html"

var appMenuItems = []components.DropdownItem{
	components.DropdownItem{
		ID:    "page1Item",
		Type:  components.MenuItem_Link,
		Label: "page1",
		Value: "page1",
	},
	components.DropdownItem{
		ID:    "page2Item",
		Type:  components.MenuItem_Link,
		Label: "page2",
		Value: "page2",
	},
	components.DropdownItem{
		Type: components.MenuItem_Divider,
	},
	components.DropdownItem{
		ID:    "editModeItem",
		Type:  components.MenuItem_Checkbox,
		Label: "Edit Mode",
		Value: "false",
	},
	components.DropdownItem{
		ID:    "aboutItem",
		Type:  components.MenuItem_Link,
		Label: "About",
		Value: "/about",
	},
}

// RenderApp renders the main application view containing the page 'pageName'
// from the URL:  /app/{pageName}
func RenderApp(w http.ResponseWriter, r *http.Request) {
	var isConnected bool = false
	//c.String(http.StatusOK, "the home page")
	siContext := r.Context().Value("session")
	if siContext != nil {
		si := siContext.(*session.SessionInfo)
		slog.Info("found session", "loginID", si.LoginID)
		connStat, _ := si.HC.ConnectionStatus()
		isConnected = connStat == transports.Connected
	}
	data := map[string]any{
		//"Title":      "HiveOT",
		"theme":      "dark",
		"theme_icon": "bi-sun", // bi-sun bi-moon-fill
		//"pages":      []string{"page1", "page2"},
		"connected": isConnected,

		// FIXME: page data. this is temporary
		"By":      "The Hive",
		"Version": "pre-alpha",
	}
	components.SetAppbarProps(data, "HiveOT", "/static/logo.svg", []string{"page1", "page2"})
	components.SetDropdownProps(data, "headerMenu", appMenuItems)

	// determine the page name from the URL and append .html if needed
	pageName := chi.URLParam(r, "pageName")
	if pageName != "" && !strings.HasSuffix(pageName, ".html") {
		pageName = pageName + ".html"
	}
	t, err := assets.AllTemplates.Clone()
	if err != nil {
		slog.Error("Can't clone templates", "err", err)
	}
	slog.Info("pagename from url", "pageName", pageName, "url", r.URL.String())
	if pageName != "" {
		pageTpl := assets.GetTemplate(pageName)
		if pageTpl == nil {
			data["appPage"] = "Error missing page template: " + pageName
		} else {
			_, err := t.AddParseTree("appPage", pageTpl.Tree)
			if err != nil {
				slog.Error("Failed adding appPage:", "err", err.Error())
			}
		}
	}
	slog.Info("RenderApp with layout", "pageName", pageName)
	assets.RenderWithLayout(w, t, templateName, "", data)

}
