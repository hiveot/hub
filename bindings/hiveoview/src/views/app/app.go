package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"net/http"
)

const AppTemplate = "app.gohtml"

// RenderApp renders the full app view without a specific page
func RenderApp(w http.ResponseWriter, r *http.Request) {

	// no specific fragment data
	data := map[string]any{}
	RenderAppPages(w, r, data)
}

// RenderAppPages renders the full html for a page in app.html.
// Instead of rendering a fragment, this renders the full application html
// with provided fragment data included.
//
// Usage: 1. Load the fragment data of the page to display
//  2. Select the app page to be viewed, whose fragment data was provided
//     e.g. append #pageName to the URL.
//  3. Render the full page with the data of the selected page.
//  4. All further navigation should render fragments using htmx.
func RenderAppPages(w http.ResponseWriter, r *http.Request, data map[string]any) {

	// 1a. full pages need the app title and header data
	if data == nil {
		data = map[string]any{}
	}
	GetAppHeadProps(data, "HiveOT", "/static/logo.svg")
	data["Status"] = GetConnectStatus(r)

	//render the full page base > app.html
	views.TM.RenderFull(w, AppTemplate, data)
}

// RenderAppOrFragment renders the page fragment, or the full app page followed
// by the fragment if this is a full page reload.
//
// A page fragment is a fragment that shows one of the main pages defined in
// app.gohtml.
//
// A full reload is needed when hx-request is not set or boost is on. In this
// case do a full page reload. For example hitting F5 page reload forces
// the browser to re-render the whole page.
//
// If hx-request is set and boost is off then just render the fragment using
// the given fragment data.
//
// app.gohtml is designed to select/render the main page fragments based on
// the url target:
//
//		<div id="directory" class="hidden" displayIfTarget="/app/directory">
//	      {{template "directory.gohtml" .}}
//	 </div>
//
// If the URL matches the target, the fragment will be included in the render
// without any data. These page fragments have a trigger that forces a fragment
// reload the first time its shown: {{$trigger := "intersect once"}}
//
// This in turns reloads the same URL as the fragment, in which case the
// go code does include the fragment data. app.go (this file) doesn't have
// to know anything about the fragment's data.
//
// pageFragment is the html file contain the fragment to render.
// data contains the page fragment data. Not used in full render.
func RenderAppOrFragment(w http.ResponseWriter, r *http.Request, pageFragment string, data any) {

	isFragment := r.Header.Get("HX-Request") != ""
	isBoosted := r.Header.Get("HX-Boosted") != ""
	if isFragment && !isBoosted {
		views.TM.RenderFragment(w, pageFragment, data)
	} else {
		// app pages fetches its own app data
		RenderAppPages(w, r, nil)
	}
}
