package main

import (
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hivehub/cmd/genvocab/vocab"
	"github.com/urfave/cli/v2"
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
		os.Exit(1)
	}
}

func GenVocab(vocabDir string, force bool) error {
	classes, constants, modTime, err := vocab.LoadVocab(vocabDir)
	if err != nil {
		//println("Error:" + err.Error())
		return err
	}
	// force using updated timestamp
	if force {
		modTime = time.Now().UTC()
	}
	err = vocab.GenVocabGo(classes, constants, modTime)
	if err != nil {
		err = fmt.Errorf("generate GO ERROR: %w", err)
		return err
	}
	err = vocab.GenVocabJS(classes, constants, modTime)
	if err != nil {
		err = fmt.Errorf("generate JS ERROR: %w", err)
		return err
	}
	err = vocab.GenVocabPy(classes, constants, modTime)
	if err != nil {
		err = fmt.Errorf("generate PY ERROR: %w", err)
		return err
	}
	return err
}
