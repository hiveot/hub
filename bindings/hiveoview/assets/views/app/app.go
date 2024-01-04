package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"net/http"
)

const templateName = "app.html"

// RenderApp renders the main application view containing the page 'pageName'
// from the URL:  /app/{pageName}
func RenderApp(w http.ResponseWriter, r *http.Request) {

	data := map[string]any{
		//"Title":      "HiveOT",
		"theme":      "dark",
		"theme_icon": "bi-sun", // bi-sun bi-moon-fill
		//"pages":      []string{"page1", "page2"},
	}
	GetAppHeadProps(data, "HiveOT", "/static/logo.svg", []string{"dashboard", "directory"})
	GetConnectStatusProps(data, r)

	assets.RenderMain(w, r, templateName, data)

}
