package assets

import (
	"errors"
	"log/slog"
	"net/http"
	"sync"
)

const baseTemplateName = "base.html"

var renderMux sync.Mutex

// RenderMain renders a main page that is directly embedded into the base html.
//
// To support progressive rendering, the base template is only used if
// the request does not have the 'HX-Request' header. This will lead to a
// full page reload.
//
// With the HX-Request header, the page is injected in the request target
// (usually the Body element) without a full page reload.
//
//	w is the writer to render to
//	name is the name of the template to render
//	data contains a map of variables to pass to the template renderer
func RenderMain(w http.ResponseWriter, r *http.Request, name string, data map[string]any) {
	isFragment := r.Header.Get("HX-Request") != ""
	if isFragment {
		RenderFragment(w, name, data)
	} else {
		RenderFull(w, name, data)
	}
}

// RenderFragment renders the template 'name' with the given data without base template.
//
//	w is the writer to render to
//	name is the name of the template to render
//	data is a map with template variables and their values
func RenderFragment(w http.ResponseWriter, name string, data map[string]any) {
	slog.Info("RenderPartial", "template", name)
	renderMux.Lock()
	defer renderMux.Unlock()
	tpl := GetTemplate(name)
	if tpl == nil || tpl.Tree == nil {
		err := errors.New("missing or invalid template: " + name)
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error(err.Error())
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	tpl, err := tpl.Clone()
	if err == nil {
		err = tpl.Execute(w, data)
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("rendering template failed", "err", err)
		_, _ = w.Write([]byte("template render error: " + err.Error()))
		return
	}
}

// RenderFull embeds the template 'name' into the base template and executes.
// The base template contains the 'embed' field where the template is injected into.
//
// If a template has an error, the error is returned to the user instead along with a 500 error.
//
//	 w is the writer to render into
//		t is the template bundle to lookup base and name
//		name is the name of the template to render
//		data contains a map of variables to pass to the template renderer
func RenderFull(w http.ResponseWriter, name string, data map[string]any) {
	slog.Info("RenderFull", "template", name)
	renderMux.Lock()
	defer renderMux.Unlock()

	baseT := AllTemplates.Lookup(baseTemplateName)
	overlayT, err := baseT.Clone()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("Cloning overlay failed", "err", err)
		_, _ = w.Write([]byte("overlay error: " + err.Error()))
		return
	}
	tpl := AllTemplates.Lookup(name)
	if tpl == nil || tpl.Tree == nil {
		err = errors.New("missing or invalid template: " + name)
	} else {
		// problem with error "cannot Clone ... after it has executed"
		tpl, err = tpl.Clone()
		// This is where the magic happens: replace the 'embed' template with the given template.
		if err == nil {
			_, err = overlayT.AddParseTree("embed", tpl.Tree)
		}
	}
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("merging templates failed", "err", err)
		_, _ = w.Write([]byte("template error: " + err.Error()))
		return
	}
	err = overlayT.Execute(w, data)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		slog.Error("rendering template failed", "err", err)
		_, _ = w.Write([]byte("template render error: " + err.Error()))
		return
	}
}

// RenderWithOverlay embeds the template 'name' into the overlay template and executes.
// The base template uses the 'embed' field.
// If a template has an error, the error is returned to the user instead along with a 500 error.
//
//	name is the name of the template to render
//	overlay is the optional overlay to use. "" for the default overlay layout.html.
//func RenderWithOverlay(w http.ResponseWriter, t *template.Template, name string, overlay string, data map[string]any) {
//	renderMux.Lock()
//	defer renderMux.Unlock()
//
//	if overlay == "" {
//		overlay = "layout.html"
//	}
//	overlayT := t.Lookup(overlay)
//	overlayT, err := overlayT.Clone()
//	if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		slog.Error("Cloning overlay failed", "err", err)
//		_, _ = w.Write([]byte("overlay error: " + err.Error()))
//		return
//	}
//	tpl := t.Lookup(name)
//	if tpl == nil || tpl.Tree == nil {
//		err = errors.New("missing or invalid template: " + name)
//	} else {
//		// problem with error "cannot Clone ... after it has executed"
//		tpl, err = tpl.Clone()
//		// This is where the magic happens: replace the 'embed' template with the given template.
//		if err == nil {
//			_, err = overlayT.AddParseTree("embed", tpl.Tree)
//		}
//	}
//	if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		slog.Error("merging templates failed", "err", err)
//		_, _ = w.Write([]byte("template error: " + err.Error()))
//		return
//	}
//	err = overlayT.Execute(w, data)
//	if err != nil {
//		w.WriteHeader(http.StatusInternalServerError)
//		slog.Error("rendering template failed", "err", err)
//		_, _ = w.Write([]byte("template render error: " + err.Error()))
//		return
//	}
//}
