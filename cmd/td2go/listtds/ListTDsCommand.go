package listtds

import (
	"fmt"
	"github.com/hiveot/hub/wot/td"
	"os"
	"path/filepath"
)

// HandleTDScan scan and display a list of TD documents
func HandleTDScan(rootDir string) error {

	fmt.Printf("Filename                             Size (KB)  Title                                @Type                   #props  #events  #actions\n")
	fmt.Printf("-----------------------------------  ---------  -----------------------------------  ---------------------      ---      ---       ---\n")

	// recursively iterate all directories looking for tdd/*.json
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		// look for tdd directories
		if d.Name() == "tdd" {
			HandleListTDs(path)
		}
		return nil
	})
	return err
}

// HandleListTDs displays a list of TD documents
func HandleListTDs(tdDir string) {

	fmt.Printf("\n%s:\n", tdDir)
	td.ForEachTD(tdDir, func(sourceFile string, tdi *td.TD) {
		stat, err := os.Stat(sourceFile)
		if err != nil {
			return
		}
		sizeKb := stat.Size() / 1024
		if err != nil {
			fmt.Printf("%-35.35s  %9d  ERROR: %s\n", stat.Name(), sizeKb, err.Error())
		} else {
			fmt.Printf("%-35.35s  %9d  %-35.35s  %-20.20s %9d %8d %9d\n", stat.Name(), sizeKb,
				tdi.Title, tdi.AtType,
				len(tdi.Properties), len(tdi.Events), len(tdi.Actions))
		}
	})

	////fmt.Printf("Filename                             Size (KB)  Title                                @Type                   #props  #events  #actions\n")
	////fmt.Printf("-----------------------------------  ---------  -----------------------------------  ---------------------      ---      ---       ---\n")
	//for _, entry := range entries {
	//	fullpath := filepath.Join(tdDir, entry.Name())
	//	tdi, err := td.ReadTD(fullpath)
	//	finfo, _ := entry.Info()
	//	sizeKb := finfo.Size() / 1024
	//	if err != nil {
	//		fmt.Printf("%-35.35s  %9d  ERROR: %s\n", entry.Name(), sizeKb, err.Error())
	//	} else {
	//		fmt.Printf("%-35.35s  %9d  %-35.35s  %-20.20s %9d %8d %9d\n", entry.Name(), sizeKb,
	//			tdi.Title, tdi.AtType,
	//			len(tdi.Properties), len(tdi.Events), len(tdi.Actions))
	//	}
	//}
	//return nil
}
