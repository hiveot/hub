package main

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"gopkg.in/yaml.v3"
	"os"
	"path"
	"strings"
	"time"
)

const golangFile = "./api/go/ht-vocab.go"
const jsFile = "./api/js/ht-vocab.js"
const pyFile = "./api/py/ht-vocab.py"
const classDir = "./api/vocab"

const ActionClassFile = "ht-action-classes.yaml"

// Generate the API source files from the vocabulary classes.
func main() {
	classes, err := LoadVocabFiles(classDir)
	if err == nil {
		lines := ExportToGolang(classes)
		data := strings.Join(lines, "\n")
		err = os.WriteFile(golangFile, []byte(data), 0664)
	}
	if err == nil {
		lines := ExportToJavascript(classes)
		data := strings.Join(lines, "\n")
		err = os.WriteFile(jsFile, []byte(data), 0664)
	}
	if err == nil {
		lines := ExportToPython(classes)
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
}

// VocabClassMap class map by vocabulary keyword
type VocabClassMap struct {
	Version   string                `yaml:"version"`
	Link      string                `yaml:"link"`
	Namespace string                `yaml:"namespace"`
	Vocab     map[string]VocabClass `yaml:"vocab"`
}

// LoadVocabFiles loads the thing, property and action classes
func LoadVocabFiles(dir string) (map[string]VocabClassMap, error) {
	classes := make(map[string]VocabClassMap)

	files, err := os.ReadDir(dir)
	if err != nil {
		return classes, err
	}
	for _, entry := range files {
		if strings.HasSuffix(entry.Name(), ".yaml") {
			vocabFile := path.Join(dir, entry.Name())
			data, err := os.ReadFile(vocabFile)
			if err == nil {
				err = yaml.Unmarshal(data, &classes)

			}
		}
	}
	return classes, err
}

// ExportToGolang writes the thing, property and action classes in a golang format
func ExportToGolang(vc map[string]VocabClassMap) []string {
	lines := make([]string, 0)

	lines = append(lines, "// Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions")
	lines = append(lines, "package vocab")
	for classType, cm := range vc {
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
	}
	return lines
}

// ExportToJavascript writes the thing, property and action classes in javascript format
func ExportToJavascript(vc map[string]VocabClassMap) []string {
	lines := make([]string, 0)

	lines = append(lines, "// Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions")
	lines = append(lines, fmt.Sprintf("// DO NOT EDIT. This file is generated and changes will be overwritten"))
	for classType, cm := range vc {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)

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
	}
	return lines
}

// ExportToPython writes the thing, property and action classes in a python format
func ExportToPython(vc map[string]VocabClassMap) []string {
	lines := make([]string, 0)

	lines = append(lines, "# Package vocab with HiveOT vocabulary names for TD Things, properties, events and actions")
	lines = append(lines, fmt.Sprintf("# DO NOT EDIT. This file is generated and changes will be overwritten"))

	for classType, cm := range vc {
		vocabKeys := utils.OrderedMapKeys(cm.Vocab)

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
	}
	return lines
}
