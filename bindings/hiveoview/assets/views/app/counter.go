package app

import (
	"github.com/hiveot/hub/bindings/hiveoview/assets"
	"net/http"
)

var counter int = 0

func RenderCounter(w http.ResponseWriter, r *http.Request) {
	counter++
	data := map[string]any{
		"value": counter,
	}
	assets.RenderTemplate(w, "counter.html", data)
}
