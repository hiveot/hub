package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/session"
	"net/http"
)

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

	sess, _, _ := session.GetSessionFromContext(r)

	data := AppHeadTemplateData{
		Ready:  true,
		Logo:   "/static/logo.svg",
		Title:  "HiveOT",
		Status: GetConnectStatus(r),
	}
	buff, err := RenderAppOrFragment(r, AppHeadTemplate, data)
	sess.WritePage(w, buff, err)
}
