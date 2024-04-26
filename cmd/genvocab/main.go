// Package main for generating vocabulary
package main

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
	"path"
	"strings"
	"time"
)

const golangFile = "./api/go/vocab/ht-vocab.go"
const jsFile = "./api/js/vocab/ht-vocab.js"
const pyFile = "./api/py/vocab/ht-vocab.py"
const classDir = "./api/vocab"

const ActionClassFile = "ht-action-classes.yaml"

// Generate the API source files from the vocabulary classes.
func main() {
	classes, constants, err := LoadVocabFiles(classDir)
	if err == nil {
		lines := ExportToGolang(classes, constants)
		data := strings.Join(lines, "\n")
		err = os.WriteFile(golangFile, []byte(data), 0664)
	}
	if err == nil {
		lines := ExportToJavascript(classes, constants)
		data := strings.Join(lines, "\n")
		err = os.WriteFile(jsFile, []byte(data), 0664)
	}
	if err == nil {
		lines := ExportToPython(classes, constants)
		data := strings.Join(lines, "\n")
		err = os.WriteFile(pyFile, []byte(data), 0664)
	}
	if err != nil {
		fmt.Println("ERROR: " + err.Error())
	}
}

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

// LoadVocabFiles loads the thing, property and action classes
func LoadVocabFiles(dir string) (map[string]VocabClassMap, map[string]VocabConstantsMap, error) {
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

// ExportToGolang writes the thing, property, action and unit classes in a golang format
func ExportToGolang(vclasses map[string]VocabClassMap, vconstants map[string]VocabConstantsMap) []string {
	fmt.Println("Generating Golang vocabulary.")
	lines := make([]string, 0)

	lines = append(lines, "// Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions")
	lines = append(lines, fmt.Sprintf("// DO NOT EDIT. This file is generated and changes will be overwritten"))
	lines = append(lines, "package vocab")

	// Loop through the vocabulary constants
	for constGroup, cm := range vconstants {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// type: %s", constGroup))
		lines = append(lines, fmt.Sprintf("// version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("// generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("// source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("// description: %s", cm.Description))
		lines = append(lines, "const (")
		for _, key := range vocabKeys {
			value := cm.Vocab[key]
			lines = append(lines, fmt.Sprintf("  %s = \"%s\"", key, value))
		}
		lines = append(lines, ")")
		lines = append(lines, "// end of "+constGroup)
	}

	// Loop through the vocabulary classes
	for classType, cm := range vclasses {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// type: %s", classType))
		lines = append(lines, fmt.Sprintf("// version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("// generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("// source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("// namespace: %s", cm.Namespace))
		lines = append(lines, "const (")
		for _, key := range vocabKeys {
			classInfo := cm.Vocab[key]
			lines = append(lines, fmt.Sprintf("  %s = \"%s\"", key, classInfo.ClassName))
		}
		lines = append(lines, ")")
		lines = append(lines, "// end of "+classType)

		//- export the map with title and description
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// %sMap maps @type to symbol, title and description", classType))
		lines = append(lines, fmt.Sprintf("var %sMap = map[string]struct {", classType))
		lines = append(lines, "   Symbol string; Title string; Description string")
		lines = append(lines, "} {")
		for key, unitInfo := range cm.Vocab {
			lines = append(lines, fmt.Sprintf(
				"  %s: {Symbol: \"%s\", Title: \"%s\", Description: \"%s\"},",
				key, unitInfo.Symbol, unitInfo.Title, unitInfo.Description))
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}

	return lines
}

// ExportToJavascript writes the thing, property and action classes in javascript format
func ExportToJavascript(vclasses map[string]VocabClassMap, vconstants map[string]VocabConstantsMap) []string {
	fmt.Println("Generating Javascript vocabulary.")
	lines := make([]string, 0)

	lines = append(lines, "// Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions")
	lines = append(lines, fmt.Sprintf("// DO NOT EDIT. This file is generated and changes will be overwritten"))

	// Loop through the vocabulary constants
	for constGroup, cm := range vconstants {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// type: %s", constGroup))
		lines = append(lines, fmt.Sprintf("// version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("// generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("// source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("// description: %s", cm.Description))
		for _, key := range vocabKeys {
			value := cm.Vocab[key]
			lines = append(lines, fmt.Sprintf("export const %s = \"%s\"", key, value))
		}
		lines = append(lines, "// end of "+constGroup)
	}

	// Loop through the types of vocabularies
	for classType, cm := range vclasses {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)

		//- export the constants
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// type: %s", classType))
		lines = append(lines, fmt.Sprintf("// version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("// generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("// source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("// namespace: %s", cm.Namespace))
		for _, key := range vocabKeys {
			classInfo := cm.Vocab[key]
			lines = append(lines, fmt.Sprintf("export const %s = \"%s\";", key, classInfo.ClassName))
		}
		lines = append(lines, "// end of "+classType)

		//- export the map with title and description
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// %sMap maps @type to symbol, title and description", classType))
		lines = append(lines, fmt.Sprintf("export const %sMap = {", classType))
		for key, unitInfo := range cm.Vocab {
			atType := cm.Vocab[key].ClassName //
			lines = append(lines, fmt.Sprintf(
				"  \"%s\": {Symbol: \"%s\", Title: \"%s\", Description: \"%s\"},",
				atType, unitInfo.Symbol, unitInfo.Title, unitInfo.Description))
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return lines
}

// ExportToPython writes the thing, property and action classes in a python format
func ExportToPython(vclasses map[string]VocabClassMap, vconstants map[string]VocabConstantsMap) []string {
	fmt.Println("Generating Python vocabulary.")
	lines := make([]string, 0)

	lines = append(lines, "# Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions")
	lines = append(lines, fmt.Sprintf("# DO NOT EDIT. This file is generated and changes will be overwritten"))

	// Loop through the vocabulary constants
	for constGroup, cm := range vconstants {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("# type: %s", constGroup))
		lines = append(lines, fmt.Sprintf("# version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("# generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("# source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("# description: %s", cm.Description))
		for _, key := range vocabKeys {
			value := cm.Vocab[key]
			lines = append(lines, fmt.Sprintf("%s = \"%s\"", key, value))
		}
		lines = append(lines, "# end of "+constGroup)
	}

	// Loop through the types of vocabularies
	for classType, cm := range vclasses {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)

		//- export the constants
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("# type: %s", classType))
		lines = append(lines, fmt.Sprintf("# version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("# generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("# source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("# namespace: %s", cm.Namespace))
		for _, key := range vocabKeys {
			classInfo := cm.Vocab[key]
			lines = append(lines, fmt.Sprintf("%s = \"%s\"", key, classInfo.ClassName))
		}
		lines = append(lines, "# end of "+classType)

		//- export the map with title and description
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("# %sMap maps @type to symbol, title and description", classType))
		lines = append(lines, fmt.Sprintf("%sMap = {", classType))
		for key, unitInfo := range cm.Vocab {
			atType := cm.Vocab[key].ClassName //
			lines = append(lines, fmt.Sprintf(
				"  \"%s\": {\"Symbol\": \"%s\", \"Title\": \"%s\", \"Description\": \"%s\"},",
				atType, unitInfo.Symbol, unitInfo.Title, unitInfo.Description))
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}
	return lines
}
