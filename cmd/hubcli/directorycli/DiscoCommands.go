package directorycli

import (
	"fmt"
	"github.com/hiveot/hub/transports/clients/discovery"
	"github.com/urfave/cli/v2"
	"time"
)

// DiscoListCommand lists discovered Thing and Directory servers
func DiscoListCommand() *cli.Command {

	return &cli.Command{
		Name: "disco",
		//Aliases: []string{"ls"},
		//ArgsUsage: "(no args)",
		Usage:    "List discovered WoT Thing and Directories",
		Category: "directory",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				return fmt.Errorf("no arguments expected")
			}
			err := HandleDiscover()
			return err
		},
	}
}

// HandleDiscover prints a list of discovered Things and Directories
func HandleDiscover() error {

	wotRecords := discovery.DiscoverTDD("", "wot", time.Second, true)
	hiveotRecords := discovery.DiscoverTDD("", "hiveot", time.Second, true)
	allRecords := append(hiveotRecords, wotRecords...)

	fmt.Println("Address                     Port  Instance       Type      Scheme   TD path")
	fmt.Println("-------                    -----  --------       ----      ------   -------")
	for _, entry := range allRecords {
		fmt.Printf("%-25s %6d  %-11s %10s   %-8s %s\n",
			entry.Addr,
			entry.Port,
			entry.Instance,
			entry.Type,
			entry.Scheme,
			entry.TD,
		)
	}
	return nil
}
