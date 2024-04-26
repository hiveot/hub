package tds

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/lib/things"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"path/filepath"
)

var tddSourceDir string = path.Join("api", "src", "tdd")

// ListTDsCommand lists the available TD documents
func ListTDsCommand() *cli.Command {
	var sourceDir string = tddSourceDir

	return &cli.Command{
		Name:  "ltd",
		Usage: "List the TD's available in the source directory",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "sources",
				Usage:       "Path to TD document sources",
				Value:       sourceDir,
				Destination: &sourceDir,
			},
		},

		Action: func(cCtx *cli.Context) error {
			err := HandleListTDs(sourceDir)
			return err
		},
	}
}

// HandleListTDs displays a list of TD documents
func HandleListTDs(sourceDir string) error {
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	fmt.Printf("Content of: %s\n", sourceDir)
	fmt.Printf("Filename              Size (KB)  Title                 @Type                 properties  events  actions\n")
	fmt.Printf("--------------------  ---------  --------------------  ---------------------        ---     ---      ---\n")
	for _, entry := range entries {
		td := things.TD{}
		fullpath := filepath.Join(sourceDir, entry.Name())
		tdJSON, err := os.ReadFile(fullpath)
		finfo, _ := entry.Info()
		sizeKb := finfo.Size() / 1024
		if err == nil {
			err = json.Unmarshal(tdJSON, &td)
		}
		if err != nil {
			fmt.Printf("%-20.20s  %9d  ERROR: %s\n", entry.Name(), sizeKb, err.Error())
		} else {
			fmt.Printf("%-20.20s  %9d  %-20.20s  %-20.20s %11d %7d %8d\n", entry.Name(), sizeKb,
				td.Title, td.AtType,
				len(td.Properties), len(td.Events), len(td.Actions))
		}
	}
	return nil
}
