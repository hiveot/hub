package genvocab

import (
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
	err := GenVocabGo(vocabDir)
	println()
	_ = GenVocabJS(vocabDir)
	println()
	_ = GenVocabPy(vocabDir)
	return err
}
