package app

import (
	"bytes"
	"github.com/hiveot/hub/services/hiveoview/src/session"
	"github.com/hiveot/hub/services/hiveoview/src/views"
	"github.com/teris-io/shortid"
	"net/http"
)

const RenderAppTemplate = "App.gohtml"

// RenderApp is used to render the full app view without a specific page
func RenderApp(w http.ResponseWriter, r *http.Request) {
	buff, err := RenderAppPages(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	} else {
		_, _ = buff.WriteTo(w)
	}
}

// RenderAppPages renders the full application html without data and injects
// a cid header field.
// Use this to load a new page or a boosted page.
// Individual pages should trigger a hx-get once they are shown to
// render themselves with data.
func RenderAppPages(r *http.Request) (buff *bytes.Buffer, err error) {

	cid := "TMP-" + shortid.MustGenerate()
	// get the connectionID to inject back into the web page hx-header field
	ctxClientSession := r.Context().Value(session.ClientSessionContextID)
	if ctxClientSession != nil {
		cs := ctxClientSession.(*session.WebClientSession)
		cid = cs.GetCID()
	}
	// This Cid field must match the one in App.gohtml hx-headers=...
	// App.gohtml:6  sse-connect="/websse?cid={{.Cid}}"
	data := map[string]string{"Cid": cid}
	return views.TM.RenderFull(RenderAppTemplate, data)
}

// RenderAppOrFragment renders the page fragment, or the full app html frame
// without data, if this is a full page reload.
//
// A fragment render is used when hx-request is set. The data needed for
// the fragment must be provided.
//
// A full reload is needed when hx-request is not set or boost is on. In this
// case do a full page reload. For example hitting F5 page reload forces
// the browser to re-render the whole page. After this each fragement will
// rerender itself with the data once they are shown.
//
// Note: App.gohtml is designed to select/render the main page fragments
// based on the url target:
//
//	<div id="directory" class="hidden" displayIfTarget="/directory">
//	      {{template "Directory.gohtml" .}}
//	</div>
//
// If the URL matches the path prefix in displayIfTarget, the fragment will
// be included in the render without any data. These page fragments have a
// trigger that forces a fragment reload the first time its shown:
//
//	{{$trigger := "intersect once"}}
//
// This in turns reloads the same URL as the fragment, in which case the
// go code does include the fragment data. app.go (this file) doesn't have
// to know anything about the fragment's data.
//
//	r is the http Request used to determine if this is a butted request or fragment
//	pageFragment is the html file contain the fragment to render.
//	data contains the page fragment data. Not used in full render.
//
// This returns the rendered page or an error if failed.
func RenderAppOrFragment(r *http.Request, pageFragment string, data any) (
	buff *bytes.Buffer, err error) {

	isFragment := r.Header.Get("HX-Request") != ""
	isBoosted := r.Header.Get("HX-Boosted") != ""

	// When the hx-boosted header is present the full page must be rendered as
	// the browser will inject the body of the rendered page into the body of
	// the browser page without doing a full page reload.
	if isFragment && !isBoosted {
		buff, err = views.TM.RenderFragment(pageFragment, data)
	} else {
		// just render the application layout without data, each page
		// element will fetch its own data.
		buff, err = RenderAppPages(r)
	}
	return buff, err
}
