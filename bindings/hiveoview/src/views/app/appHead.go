package app

import "net/http"

const AppHeadTemplate = "appHead.gohtml"

//const AppMenuTemplate = "appMenu.gohtml"

type AppHeadTemplateData struct {
	Ready  bool
	Logo   string
	Title  string
	Status *ConnectStatus
}

// RenderAppHead renders the app header fragment
func RenderAppHead(w http.ResponseWriter, r *http.Request) {

	data := AppHeadTemplateData{
		Ready:  true,
		Logo:   "/static/logo.svg",
		Title:  "HiveOT",
		Status: GetConnectStatus(r),
	}
	RenderAppOrFragment(w, r, AppHeadTemplate, data)
}
