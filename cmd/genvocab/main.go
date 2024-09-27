package main

import (
	"github.com/hiveot/hub/cmd/genvocab/vocab"
	"path"
)

const Version = `0.1-alpha`

// CLI for generating vocabulary out of yaml file
func main() {
	vocabDir := path.Join("api", "src", "vocab")

	vocab.GenVocab(vocabDir)
}
