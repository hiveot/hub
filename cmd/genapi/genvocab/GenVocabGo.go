// Package genvocab for generating vocabulary
package genvocab

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"os"
	"strings"
	"time"
)

const golangFile = "./api/go/vocab/ht-vocab.go"

// GenVocabGo generates the vocabulary constants in golang.
func GenVocabGo(sourceDir string) error {
	classes, constants, err := LoadVocab(sourceDir)
	if err == nil {
		lines := ExportToGolang(classes, constants)
		data := strings.Join(lines, "\n")
		println("Writing: " + golangFile)
		err = os.WriteFile(golangFile, []byte(data), 0664)
	}
	if err != nil {
		fmt.Println("ERROR: " + err.Error())
	}
	return err
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
