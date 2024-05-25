package services

import (
	"encoding/json"
	"fmt"
	_go "github.com/hiveot/hub/cmd/genapi/services/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// GenGoAPICommand is a golang API generator from TD documents
func GenGoAPICommand() *cli.Command {
	var sourceDir string = tddSourceDir
	var outDir string = path.Join("api", "go")
	return &cli.Command{
		Name:  "ggo",
		Usage: "Generate a Go API of the TDs in the sources directory",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "outdirbase",
				Usage:       "Base path for packages",
				Value:       outDir,
				Destination: &outDir,
			},
			&cli.StringFlag{
				Name:        "sources",
				Usage:       "Path to TD document sources",
				Value:       sourceDir,
				Destination: &sourceDir,
			},
		},
		Action: func(cCtx *cli.Context) error {
			err := HandleGenGoAPI(sourceDir, outDir)
			return err
		},
	}
}

// HandleGenGoAPI generates a go API for the .JSON sources
func HandleGenGoAPI(sourceDir string, outDirBase string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	fmt.Printf("Source directory: %s\n", sourceDir)
	fmt.Printf("Destination base directory: %s\n", outDirBase)
	fmt.Printf("Source file           Size (KB)  Title                 Output                                    Status\n")
	fmt.Printf("--------------------  ---------  --------------------  ----------------------------------------  ------\n")
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
		}

		td := things.TD{}
		fullpath := filepath.Join(sourceDir, entry.Name())
		tdJSON, err := os.ReadFile(fullpath)
		sizeKb := finfo.Size() / 1024
		if err == nil {
			err = json.Unmarshal(tdJSON, &td)
		}
		outputStatus := "Failed"
		outfile := ""
		if err == nil {
			// the source filename will be the package name
			noext := name[:len(name)-len(ext)]
			outDir := path.Join(outDirBase, noext)
			_ = os.MkdirAll(outDir, 0755)
			outfile = path.Join(outDir, noext+".go")
			err = _go.GenGoAPIFromTD(&td, outfile)
		}
		if err == nil {
			outputStatus = "Success"
		}

		if err != nil {
			fmt.Printf("%-20.20s  %9d  ERROR: %s\n",
				entry.Name(), sizeKb, err.Error())
		} else {
			fmt.Printf("%-20.20s  %9d  %-20.20s  %-40.40s  %s\n",
				entry.Name(), sizeKb, td.Title, outfile, outputStatus)
		}
	}
	return nil
}
