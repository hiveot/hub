package about

import (
	"github.com/hiveot/hub/services/hiveoview/src/views/app"
	"net/http"
)

const TemplateFile = "RenderAboutPage.gohtml"

type AboutPageTemplateData struct {
	// HiveOT version
	Version string
}

func RenderAboutPage(w http.ResponseWriter, r *http.Request) {
	data := &AboutPageTemplateData{
		Version: "Early Alpha",
	}
	buff, err := app.RenderAppOrFragment(r, TemplateFile, data)
	_ = err
	buff.WriteTo(w)
}