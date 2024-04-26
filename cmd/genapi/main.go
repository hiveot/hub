package main

import (
	"github.com/hiveot/hub/cmd/genapi/goapi"
	"github.com/hiveot/hub/cmd/genapi/tds"
	"github.com/hiveot/hub/lib/logging"
	"github.com/urfave/cli/v2"
	"os"
)

const Version = `0.1-alpha`

// CLI for generating API's from Thing Description Documents (TD)
func main() {
	var sourceDir string = "api/tdd"
	logging.SetLogging("warning", "")

	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "genapi",
		Usage:                "HiveOT API Generator from WoT TD documents",
		Version:              Version,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "sources",
				Usage:       "Path to TD document sources",
				Value:       sourceDir,
				Destination: &sourceDir,
			},
		},
		// commands arguments are passed by reference so they are updated in the Before section
		Commands: []*cli.Command{
			tds.ListTDsCommand(sourceDir),
			goapi.GenGoAPICommand(sourceDir),
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
