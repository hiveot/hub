package src

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/cmd/tdd2api/src/tdd2go"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/tdd"
	jsoniter "github.com/json-iterator/go"
	"github.com/urfave/cli/v2"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// Tdd2GoCommand is a golang API generator from TD documents
func Tdd2GoCommand(apiDir string) *cli.Command {
	var sourceDir string = tddSourceDir
	var outDir string = path.Join(apiDir, "go")
	return &cli.Command{
		Name:  "ggo",
		Usage: "Generate a Go API of the TDs in the sources directory",

		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() == 0 {
				return fmt.Errorf("Expected a directory with .json source files")
			}
			sourceDir = cCtx.Args().First()
			err := HandleTdd2Go(sourceDir, outDir)
			return err
		},
	}
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

// HandleTdd2Go generates a go API for the .JSON sources
func HandleTdd2Go(sourceDir string, outDirBase string) error {
	sourceFiles, err := GetSourceFilesInDir(sourceDir)
	if err != nil {
		return err
	}
	fmt.Printf("Source directory: %s\n", sourceDir)
	//fmt.Printf("Destination base directory: %s\n", outDirBase)
	fmt.Printf("Source file               Size (KB)  ThingID                     Title                           Output                                    Status\n")
	fmt.Printf("------------------------  ---------  --------------------------  ------------------------------  ----------------------------------------  ------\n")
	for _, fullPath := range sourceFiles {
		td := tdd.TD{}
		tdJSON, err := os.ReadFile(fullPath)
		sizeKb := len(tdJSON) / 1024
		if err == nil {
			err = jsoniter.Unmarshal(tdJSON, &td)
			if err != nil {
				err = fmt.Errorf("Unmarshal error in file '%s': %w",
					fullPath, err)
				slog.Error(err.Error())
			}
		}
		outputStatus := "Failed"
		_, sourceFile := path.Split(fullPath)
		sourceExt := path.Ext(sourceFile)
		sourceNoExt := sourceFile[:(len(sourceFile) - len(sourceExt))]
		if td.AtContext == nil {
			var outFile string
			var packageName string
			var typeName string
			// not a TD, assume this is a standalone dataschema
			var ds tdd.DataSchema
			err = json.Unmarshal(tdJSON, &ds)

			// FIXME: using @type as package/type name is an experiment
			atType := ds.GetAtTypeString()
			if atType != "" {
				parts := strings.Split(atType, "/")
				packageName = parts[0]
				if len(parts) > 1 {
					outFile = path.Join(outDirBase, atType+".go")
					typeName = parts[1]
				}
			} else {
				// default package
				packageName = "hub"
				typeName = sourceNoExt
			}
			outFile = path.Join(outDirBase, packageName, typeName+".go")
			l := &utils.SL{}
			idTitle := tdd2go.ToTitle(typeName)
			l.Add("package " + packageName)
			l.Add("")
			tdd2go.GenSchemaDefStruct(l, sourceNoExt, idTitle, &ds)
			err = l.Write(outFile)

			break
		}

		agentID, _ := tdd.SplitDigiTwinThingID(td.ID)
		outFile := path.Join(outDirBase, agentID, sourceNoExt+".go")

		if err == nil {
			_ = os.MkdirAll(outDirBase, 0755)
			err = tdd2go.GenGoAPIFromTD(&td, outFile)
		}
		if err == nil {
			outputStatus = "Success"
		}
		if err != nil {
			fmt.Printf("%-20.20s  %9d  ERROR: %s\n",
				sourceFile, sizeKb, err.Error())
		} else {
			fmt.Printf("%-24.24s  %9d  %-26.26s  %-30.30s  %-40.40s  %s\n",
				sourceFile, sizeKb, td.ID, td.Title, outFile, outputStatus)
		}
	}
	return nil
}
