package src

import (
	"github.com/hiveot/hub/cmd/tdd2api/src/vocab"
	"github.com/urfave/cli/v2"
	"path"
)

// GenVocabCommand generates vocabulary
func GenVocabCommand() *cli.Command {
	var vocabDir = path.Join("api", "src", "vocab")

	return &cli.Command{
		Name:  "vocab",
		Usage: "Generate the vocabulary",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "sources",
				Usage:       "Path to vocab sources",
				Value:       vocabDir,
				Destination: &vocabDir,
			},
		},
		Action: func(cCtx *cli.Context) error {
			err := GenVocab(vocabDir)
			return err
		},
	}
}

func GenVocab(vocabDir string) error {
	err := vocab.GenVocabGo(vocabDir)
	println()
	_ = vocab.GenVocabJS(vocabDir)
	println()
	_ = vocab.GenVocabPy(vocabDir)
	return err
}
