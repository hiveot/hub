package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hiveot/hub/cmd/genvocab/vocab"
	"github.com/urfave/cli/v3"
)

const Version = `0.1-alpha`

// [Deprecated] CLI for generating vocabulary out of yaml file
// genvocab should convert the golang source file to js and puthon
func main() {
	var vocabDir = path.Join("api", "src", "vocab")
	var force bool

	app := &cli.Command{
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
		Action: func(ctx context.Context, cmd *cli.Command) error {
			err := GenVocab(vocabDir, force)
			return err
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
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
