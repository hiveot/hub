package goapi

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// GenGoAPICommand is a golang API generator from TD documents
func GenGoAPICommand(sourceDir string) *cli.Command {
	var tmplDir string
	var outDir string
	return &cli.Command{
		Name:  "ggo",
		Usage: "Generate a Go API of the TDs in the sources directory",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "templates",
				Usage:       "Path with templates folder",
				Value:       tmplDir,
				Destination: &tmplDir,
			},
			&cli.StringFlag{
				Name:        "outdir",
				Usage:       "Path to output directory",
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
			err := HandleGenGoAPI(sourceDir, tmplDir, outDir)
			return err
		},
	}
}

// HandleGenGoAPI generates a go API for the .JSON sources
func HandleGenGoAPI(sourceDir string, tmplDir string, outDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	fmt.Printf("Content of: %s\n", sourceDir)
	fmt.Printf("Filename              Size (KB)  Title                 Output\n")
	fmt.Printf("--------------------  ---------  --------------------  ------\n")
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
		if err == nil {
			// replace the extension with '.go'
			noext := name[:len(name)-len(ext)]
			outfile := path.Join(outDir, noext+".go")
			err = GenGoAPI(&td, outfile, tmplDir)
		}
		if err == nil {
			outputStatus = "Success"
		}

		if err != nil {
			fmt.Printf("%-20.20s  %9d  ERROR: %s\n",
				entry.Name(), sizeKb, err.Error())
		} else {
			fmt.Printf("%-20.20s  %9d  %-20.20s  %s\n",
				entry.Name(), sizeKb, td.Title, outputStatus)
		}
	}
	return nil
}

func GenGoAPI(td *things.TD, outfile string, tmplDir string) (err error) {
	l := L{}
	l = l.Add("package %s", td.ID)
	l = l.Add("")
	l = l.Add("import \"github.com/hiveot/hub/runtime/api\"")
	l = l.Add("")
	l = GenActionStructs(l, td)
	l = GenActionClient(l, td)
	//l = GenActionHandler(l, td)

	if len(l) > 0 {
		err = l.Write(outfile)
	}
	return err
}
