package src

import (
	"fmt"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"github.com/urfave/cli/v2"
	"os"
	"path"
	"path/filepath"
)

var tddSourceDir string = path.Join("api", "src", "tdd")

// ListTDsCommand lists the available TD documents
func ListTDsCommand() *cli.Command {

	return &cli.Command{
		Name:  "ltd",
		Usage: "List the TD's available in the source directory",

		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() == 0 {
				return fmt.Errorf("Missing source directory")
			}
			sourceDir := cCtx.Args().First()
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
		td := td.TD{}
		fullpath := filepath.Join(sourceDir, entry.Name())
		tdJSON, err := os.ReadFile(fullpath)
		finfo, _ := entry.Info()
		sizeKb := finfo.Size() / 1024
		if err == nil {
			err = jsoniter.Unmarshal(tdJSON, &td)
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
