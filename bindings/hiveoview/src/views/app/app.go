package app

import (
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"net/http"
	"strings"
)

const templateName = "app.html"

// RenderApp renders the main application view containing the page 'pageName'
// from the URL:  /app/#{pageName}
func RenderApp(w http.ResponseWriter, r *http.Request) {

	data := map[string]any{
		//"Title":      "HiveOT",
		"theme":      "dark",
		"theme_icon": "bi-sun", // bi-sun bi-moon-fill
		//"pages":      []string{"page1", "page2"},
	}
	GetAppHeadProps(data, "HiveOT", "/static/logo.svg", []string{"dashboard", "directory"})
	GetConnectStatusProps(data, r)

	views.TM.RenderTemplate(w, r, templateName, data)

}

// RenderAppPage renders the app sub-page injected into the app page html
// This is invoked by the app page on load.
// This supports both full page reloads and htmx fragments
func RenderAppPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{
		//"Title":      "HiveOT",
		"theme":      "dark",
		"theme_icon": "bi-sun", // bi-sun bi-moon-fill
		//"pages":      []string{"page1", "page2"},
	}
	GetAppHeadProps(data, "HiveOT", "/static/logo.svg", []string{"dashboard", "directory"})

	// determine the page name from the URL and append .html if needed
	pageName := chi.URLParam(r, "pageName")
	if pageName != "" && !strings.HasSuffix(pageName, ".html") {
		pageName = pageName + ".html"
	} else if pageName == "" {
		pageName = "status.html"
	}
	data["pageName"] = pageName
	//views.TM.RenderTemplate(w, r, pageName, data)
	//slog.Info("pagename from url", "pageName", pageName, "url", r.URL.String())
	isHtmx := r.Header.Get("HX-Request") != ""
	if isHtmx {
		// render the partial
		views.TM.RenderFragment(w, pageName, data)
		//	//fmt.Fprintf(w, "<div>Hello "+pageName+"</div>")
	} else {
		//render the full page base>app>page
		// FIXME-1: get proper app data
		//slog.Info("RenderFull App with trigger of loading the page", "pageName", pageName)
		views.TM.RenderFull(w, "app.html", data)
		//
		//	// ? Include a htmx trigger in the response that loads the app page as a partial?
		//	// this would speed up initial page load
		//	//pageTpl := assets.GetTemplate(pageName)
		//	//assets.RenderPartial(w, pageName, data)
	}

}
