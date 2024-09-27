package main

import (
	"github.com/hiveot/hub/cmd/tdd2api/src"
	"github.com/hiveot/hub/lib/logging"
	"github.com/urfave/cli/v2"
	"os"
)

const Version = `0.1-alpha`

// CLI for generating API's from Thing Description Documents (TD)
func main() {
	logging.SetLogging("warning", "")
	var apiDir = "./api"

	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "tdd2api",
		Usage:                "HiveOT API Generator from TD document definitions",
		Version:              Version,
		// commands arguments are passed by reference so they are updated in the Before section
		Commands: []*cli.Command{
			//src.GenVocabCommand(),
			src.ListTDsCommand(),
			src.Tdd2GoCommand(apiDir),
		},
	}
	app.Suggest = true
	app.HideHelpCommand = true
	if err := app.Run(os.Args); err != nil {
		println("ERROR: ", err.Error())
		//helpArgs := append(os.Args, "-h")
		//_ = app.Run(helpArgs)
	}
}
