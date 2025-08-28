package listtms

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hiveot/hub/wot/td"
)

// HandleTMScan scan and display a list of TM (TD model) documents
func HandleTMScan(rootDir string) error {

	fmt.Printf("Filename                             Size (KB)  Title                                @Type                   #props  #events  #actions\n")
	fmt.Printf("-----------------------------------  ---------  -----------------------------------  ---------------------      ---      ---       ---\n")

	// recursively iterate all directories looking for tm/*.json
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		// look for tdd directories
		if d.Name() == "tm" {
			HandleListTMs(path)
		}
		return nil
	})
	return err
}

// HandleListTMs displays a list of TM documents
func HandleListTMs(tmDir string) {

	fmt.Printf("\n%s:\n", tmDir)
	td.ForEachTD(tmDir, func(sourceFile string, tmDoc *td.TD) {
		stat, err := os.Stat(sourceFile)
		if err != nil {
			return
		}
		sizeKb := stat.Size() / 1024
		if err != nil {
			fmt.Printf("%-35.35s  %9d  ERROR: %s\n", stat.Name(), sizeKb, err.Error())
		} else {
			fmt.Printf("%-35.35s  %9d  %-35.35s  %-20.20s %9d %8d %9d\n", stat.Name(), sizeKb,
				tmDoc.Title, tmDoc.AtType,
				len(tmDoc.Properties), len(tmDoc.Events), len(tmDoc.Actions))
		}
	})

	////fmt.Printf("Filename                             Size (KB)  Title                                @Type                   #props  #events  #actions\n")
	////fmt.Printf("-----------------------------------  ---------  -----------------------------------  ---------------------      ---      ---       ---\n")
	//for _, entry := range entries {
	//	fullpath := filepath.Join(tmDir, entry.Name())
	//	tdi, err := td.RetrieveThing(fullpath)
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
