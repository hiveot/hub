package app

import (
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"log/slog"
	"net/http"
	"strings"
)

func RenderAppPage(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	// determine the page name from the URL and append .html if needed
	pageName := chi.URLParam(r, "pageName")
	if pageName != "" && !strings.HasSuffix(pageName, ".html") {
		pageName = pageName + ".html"
	}
	data["pageName"] = pageName
	slog.Info("pagename from url", "pageName", pageName, "url", r.URL.String())
	isHtmx := r.Header.Get("HX-Request") != ""
	// render the page as-is
	if isHtmx {
		// render the partial
		//assets.RenderTemplate(w, pageName, data)
		fmt.Fprintf(w, "<div>Hello "+pageName+"</div>")
	} else {
		//render the full page
		// FIXME: get proper app data
		slog.Info("RenderApp with layout", "pageName", pageName)
		pageTpl := assets.GetTemplate(pageName)
		assets.RenderWithLayout(w, pageTpl, pageName, "", data)
	}

}
