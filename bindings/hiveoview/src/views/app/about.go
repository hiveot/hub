package app

import (
	"net/http"
)

const TemplateFile = "aboutPage.gohtml"

func RenderAbout(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	buff, err := RenderAppOrFragment(r, TemplateFile, data)
	_ = err
	buff.WriteTo(w)
}
