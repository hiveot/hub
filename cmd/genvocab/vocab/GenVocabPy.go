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

const pyFile = "./api/py/vocab/vocab.py"

// GenVocabPy generates the vocabulary constants in python.
func GenVocabPy(
	classes map[string]VocabClassMap,
	constants map[string]VocabConstantsMap,
	modTime time.Time) error {

	err := os.MkdirAll(path.Dir(pyFile), 0755)
	if err != nil {
		return err
	}
	outfileStat, err := os.Stat(pyFile)
	if err == nil && outfileStat.ModTime().After(modTime) {
		fmt.Printf("GenVocabPY: Destination '%s' is already up to date, not updating\n", pyFile)
		return nil
	}
	lines := ExportToPython(classes, constants)
	data := strings.Join(lines, "\n")
	println("Writing: " + pyFile)
	err = os.WriteFile(pyFile, []byte(data), 0664)
	return err
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
