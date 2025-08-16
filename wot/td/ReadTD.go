package td

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// ForEachTD iterate the given TD folder and invokes the handler with the TD instance
func ForEachTD(tdDir string, handler func(sourceFile string, tdi *TD)) {
	sourceFiles, err := GetSourceFilesInDir(tdDir)
	if err != nil {
		return
	}
	for _, sourceFile := range sourceFiles {
		// 1: load the TD
		tdi, err := ReadTD(sourceFile)
		if err != nil {
			fmt.Printf("file '%s' is not a valid TD JSON file: %s", sourceFile, err)
		} else {
			// Invoke the handler
			handler(sourceFile, tdi)
		}
	}
	return
}

// GetSourceFilesInDir return the list of .json source files in the given directory
func GetSourceFilesInDir(sourceDir string) ([]string, error) {
	sourceFiles := make([]string, 0)
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return sourceFiles, err
	}
	for _, entry := range entries {
		finfo, _ := entry.Info()
		name := finfo.Name()
		ext := path.Ext(name)
		if entry.IsDir() {
			continue
		}
		// filter on .json files
		if strings.ToLower(ext) != ".json" {
			continue
		} else {
			//fmt.Printf("Adding %s\n", name)
			fullPath := filepath.Join(sourceDir, entry.Name())
			sourceFiles = append(sourceFiles, fullPath)
		}
	}
	return sourceFiles, nil
}

// ReadTD returns the TD instance of a TM/TD loaded from file
//
//	sourceFile is the file containing the TM/TD in JSON
func ReadTD(sourceFile string) (*TD, error) {
	tdi := TD{}
	tdJSON, err := os.ReadFile(sourceFile)
	if err == nil {
		// json has better error reporting
		err = json.Unmarshal(tdJSON, &tdi)
	}
	if err != nil {
		err = fmt.Errorf("ReadTD failed for file '%s': %w", sourceFile, err)
	}
	return &tdi, err
}
