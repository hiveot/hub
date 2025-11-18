package about

import (
	"github.com/hiveot/hivehub/services/hiveoview/src/views/app"
	"net/http"
)

const RenderAboutTemplate = "AboutPage.gohtml"

type AboutPageTemplateData struct {
	// HiveOT version
	Version string
}

func RenderAboutPage(w http.ResponseWriter, r *http.Request) {
	data := &AboutPageTemplateData{
		Version: "Early Alpha",
	}
	buff, err := app.RenderAppOrFragment(r, RenderAboutTemplate, data)
	_ = err
	buff.WriteTo(w)
}
