package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/views"
	"net/http"
)

const templateName = "app.html"

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
	GetConnectStatusProps(data, r)

	//render the full page base > app.html
	views.TM.RenderFull(w, "app.html", data)
}

// RenderAppOrFragment renders the full html for a page in app.html using the given data,
// or if hx-request is set and hx-boost is not, just the fragment.
//
// This is a one-stop shop for rendering a page both as a fragment or a full reload (from a bookmark)
//
// This just renders the page, it does not select it for viewing (which is a client side concern)
//
// fragmentHtml is the html file contain the fragment to render.
// data contains the fragment data, also used in full render.
func RenderAppOrFragment(w http.ResponseWriter, r *http.Request, fragmentHtml string, data map[string]any) {

	isFragment := r.Header.Get("HX-Request") != ""
	isBoosted := r.Header.Get("HX-Boosted") != ""
	if isFragment && !isBoosted {
		views.TM.RenderFragment(w, fragmentHtml, data)
	} else {
		RenderAppPages(w, r, data)
	}
}
