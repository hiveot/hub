package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"net/http"
)

const AppTemplate = "app.gohtml"

// RenderApp renders the full app view without a specific page
func RenderApp(w http.ResponseWriter, r *http.Request) {

	// no specific fragment data
	RenderAppPages(w, r, nil)
}

// RenderAppPages renders the full html for a page in app.html.
// Use this to load a new page or a boosted page.
// data is the data expected for the page to load. Use nil to default to app head
// containing the connect status for the header.
func RenderAppPages(w http.ResponseWriter, r *http.Request, data any) {

	//render the full page base > app.html
	// Data can be the current app page
	// data must contain the app header properties, eg connection status
	//views.TM.RenderFull(w, AppTemplate, data)
	views.TM.RenderFull(w, AppTemplate, nil)
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
	// When the hx-boosted header is present the full page must be rendered as
	// the browser will inject the body of the rendered page into the body of
	// the browser page without doing a full page reload.
	if isFragment && !isBoosted {
		views.TM.RenderFragment(w, pageFragment, data)
	} else {
		// app pages fetches its own app data
		RenderAppPages(w, r, data)
	}
}
