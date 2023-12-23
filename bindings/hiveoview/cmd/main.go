package main

import (
	"github.com/hiveot/hub/bindings/hiveoview/src/service"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/plugin"
)

const port = 8080 // test port

// During development, run with 'air' and set home to the hiveoview source directory.
// eg, from the hiveoview directory: 'air ./cmd/main.go --home ./hiveoview'
// - air will rebuild and relaunch hiveoview on file changes
// - hiveoview will monitor the templates folder and notify connected web browsers on changes
func main() {

	env := plugin.GetAppEnvironment("", true)
	_ = env
	logging.SetLogging(env.LogLevel, "")
	// TODO: get port and debug from config

	// serve the hiveoview web pages
	svc := service.NewHiveovService(port, false)
	svc.Start()

	plugin.WaitForSignal()
}
