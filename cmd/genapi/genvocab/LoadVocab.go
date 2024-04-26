// Package genvocab for generating vocabulary
package genvocab

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
	"strings"
)

const ActionClassFile = "ht-action-classes.yaml"

// VocabClass holds a vocabulary entry
type VocabClass struct {
	ClassName   string `yaml:"class"`
	Title       string `yaml:"title"`
	Description string `yaml:"description"`
	Symbol      string `yaml:"symbol,omitempty"` // for units
}

// VocabClassMap class map by vocabulary keyword
type VocabClassMap struct {
	Version     string                `yaml:"version"`
	Link        string                `yaml:"link"`
	Namespace   string                `yaml:"namespace"`
	Description string                `yaml:"description"`
	Vocab       map[string]VocabClass `yaml:"vocab"`
}

// VocabConstantsMap map of application constants
type VocabConstantsMap struct {
	Version     string            `yaml:"version"`
	Link        string            `yaml:"link"`
	Namespace   string            `yaml:"namespace"`
	Description string            `yaml:"description"`
	Vocab       map[string]string `yaml:"vocab"`
}

// LoadVocab loads the thing, property and action vocabulary classes
func LoadVocab(dir string) (map[string]VocabClassMap, map[string]VocabConstantsMap, error) {
	vocabClasses := make(map[string]VocabClassMap)
	vocabConstants := make(map[string]VocabConstantsMap)

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	for _, entry := range files {
		vocabFile := path.Join(dir, entry.Name())

		data, err := os.ReadFile(vocabFile)
		if err == nil {
			if strings.HasSuffix(entry.Name(), "classes.yaml") {
				fmt.Println("Reading vocab classes from " + vocabFile)
				err = yaml.Unmarshal(data, &vocabClasses)
			} else if strings.HasSuffix(entry.Name(), ".yaml") {
				fmt.Println("Reading vocab constants from " + vocabFile)
				err = yaml.Unmarshal(data, &vocabConstants)
			} else {
				slog.Error("Ignored non-yaml file: " + vocabFile)
			}
		} else {
			slog.Error("Error reading " + vocabFile + ": " + err.Error())
		}
	}
	return vocabClasses, vocabConstants, err
}
