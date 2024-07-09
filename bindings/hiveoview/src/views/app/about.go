package app

import (
	"net/http"
)

const TemplateFile = "aboutPage.gohtml"

func RenderAbout(w http.ResponseWriter, r *http.Request) {
	data := map[string]any{}
	RenderAppOrFragment(w, r, TemplateFile, data)
}
