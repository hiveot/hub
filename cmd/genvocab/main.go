package main

import (
	"github.com/hiveot/hub/cmd/genvocab/vocabs"
	"github.com/hiveot/hub/lib/logging"
	"github.com/urfave/cli/v2"
	"os"
)

const Version = `0.1-alpha`

// CLI for generating WoT and HiveOT vocabulary constants
func main() {
	logging.SetLogging("warning", "")

	app := &cli.App{
		EnableBashCompletion: true,
		Name:                 "genvocab",
		Usage:                "HiveOT Vocabulary code generator",
		Version:              Version,

		// commands arguments are passed by reference so they are updated in the Before section
		Commands: []*cli.Command{
			vocabs.GenVocabCommand(),
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
