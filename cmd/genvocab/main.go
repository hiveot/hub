package main

import (
	"fmt"
	"github.com/hiveot/hub/cmd/genvocab/vocab"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"time"
)

const Version = `0.1-alpha`

// CLI for generating vocabulary out of yaml file
func main() {
	var vocabDir = path.Join("api", "src", "vocab")
	var force bool

	app := &cli.App{
		Name:  "vocab",
		Usage: "Generate the vocabulary",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "sources",
				Usage:       "Path to vocab sources",
				Value:       vocabDir,
				Destination: &vocabDir,
			},
			&cli.BoolFlag{
				Name:        "force",
				Usage:       "Overwite existing sources",
				Value:       force,
				Destination: &force,
				Aliases:     []string{"f"},
			},
		},
		Action: func(cCtx *cli.Context) error {
			err := GenVocab(vocabDir, force)
			return err
		},
	}
	if err := app.Run(os.Args); err != nil {
		println("ERROR: ", err.Error())
	}
}

func GenVocab(vocabDir string, force bool) error {
	classes, constants, modTime, err := vocab.LoadVocab(vocabDir)
	if err != nil {
		println("Error:" + err.Error())
		return err
	}
	// force using updated timestamp
	if force {
		modTime = time.Now().UTC()
	}
	err = vocab.GenVocabGo(classes, constants, modTime)
	if err != nil {
		fmt.Println("Gen GO ERROR: " + err.Error())
	}
	err = vocab.GenVocabJS(classes, constants, modTime)
	if err != nil {
		fmt.Println("Gen JS ERROR: " + err.Error())
	}
	err = vocab.GenVocabPy(classes, constants, modTime)
	if err != nil {
		fmt.Println("Gen PY ERROR: " + err.Error())
	}
	return err
}
