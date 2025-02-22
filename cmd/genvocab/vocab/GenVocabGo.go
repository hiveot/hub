// Package genvocab for generating vocabulary
package vocab

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"os"
	"path"
	"strings"
	"time"
)

const golangVocabFile = "./api/go/vocab/vocab.go"

// GenVocabGo generates the vocabulary constants in golang.
func GenVocabGo(classes map[string]VocabClassMap,
	constants map[string]VocabConstantsMap,
	modTime time.Time) error {

	err := os.MkdirAll(path.Dir(golangVocabFile), 0755)
	if err != nil {
		return err
	}
	outfileStat, err := os.Stat(golangVocabFile)
	if err == nil && outfileStat.ModTime().After(modTime) {
		fmt.Printf("GenVocabGo: Destination '%s' is already up to date, not updating\n", golangVocabFile)
		return nil
	}
	lines := ExportToGolang(classes, constants)
	data := strings.Join(lines, "\n")
	println("Writing: " + golangVocabFile)
	err = os.WriteFile(golangVocabFile, []byte(data), 0664)

	return err
}

// ExportToGolang writes the thing, property, action and unit classes in a golang format
func ExportToGolang(vclasses map[string]VocabClassMap, vconstants map[string]VocabConstantsMap) []string {
	fmt.Println("Generating Golang vocabulary.")
	lines := make([]string, 0)

	lines = append(lines, "// Package vocab with HiveOT and WoT vocabulary names for TD Things, properties, events and actions")
	lines = append(lines, fmt.Sprintf("// DO NOT EDIT. This file is generated and changes will be overwritten"))
	lines = append(lines, "package vocab")

	// Loop through the vocabulary constants
	for constGroup, cm := range vconstants {
		vocabNames := utils.OrderedMapKeys(cm.Vocab)
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// type: %s", constGroup))
		lines = append(lines, fmt.Sprintf("// version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("// generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("// source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("// description: %s", cm.Description))
		lines = append(lines, "const (")
		for _, name := range vocabNames {
			value := cm.Vocab[name]
			lines = append(lines, fmt.Sprintf("  %s = \"%s\"", name, value))
		}
		lines = append(lines, ")")
		lines = append(lines, "// end of "+constGroup)
	}

	// Loop through the vocabulary classes
	for classType, cm := range vclasses {
		vocabNames := utils.OrderedMapKeys(cm.Vocab)

		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// type: %s", classType))
		lines = append(lines, fmt.Sprintf("// version: %s", cm.Version))
		lines = append(lines, fmt.Sprintf("// generated: %s", time.Now().Format(time.RFC822)))
		lines = append(lines, fmt.Sprintf("// source: %s", cm.Link))
		lines = append(lines, fmt.Sprintf("// namespace: %s", cm.Namespace))
		lines = append(lines, "const (")
		for _, name := range vocabNames {
			classInfo := cm.Vocab[name]
			lines = append(lines, fmt.Sprintf("  %s = \"%s\"", name, classInfo.ClassName))
		}
		lines = append(lines, ")")
		lines = append(lines, "// end of "+classType)

		//- export the map with title and description
		lines = append(lines, "")
		lines = append(lines, fmt.Sprintf("// %sMap maps @type to symbol, title and description", classType))
		lines = append(lines, fmt.Sprintf("var %sMap = map[string]struct {", classType))
		lines = append(lines, "   Symbol string; Title string; Description string")
		lines = append(lines, "} {")
		for name, unitInfo := range cm.Vocab {
			lines = append(lines, fmt.Sprintf(
				"  %s: {Symbol: \"%s\", Title: \"%s\", Description: \"%s\"},",
				name, unitInfo.Symbol, unitInfo.Title, unitInfo.Description))
		}
		lines = append(lines, "}")
		lines = append(lines, "")
	}

	return lines
}
