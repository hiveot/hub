package directorycli

import (
	"encoding/json"
	"fmt"
	"github.com/hiveot/hub/transports/clients/discovery"
	"github.com/hiveot/hub/transports/tputils/tlsclient"
	jsoniter "github.com/json-iterator/go"
	"github.com/urfave/cli/v2"
	"time"
)

// DiscoListCommand lists discovered Thing and Directory servers
//
// authToken used to read the TD
func DiscoListCommand(authToken *string) *cli.Command {
	var readtd = false
	return &cli.Command{
		Name: "disco",
		//Aliases: []string{"ls"},
		ArgsUsage: "[--td]",
		Usage:     "List discovered WoT Thing and Directories",
		Category:  "directory",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "td",
				Usage:       "Read the directory TD",
				Value:       false,
				Destination: &readtd,
			},
		}, Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				return fmt.Errorf("no arguments expected")
			}
			err := HandleDiscover(readtd, *authToken)
			return err
		},
	}
}

// HandleDiscover prints a list of discovered Things and Directories
func HandleDiscover(readtd bool, authToken string) error {

	wotRecords := discovery.DiscoverTDD("", "wot", time.Second, true)
	hiveotRecords := discovery.DiscoverTDD("", "hiveot", time.Second, true)
	allRecords := append(hiveotRecords, wotRecords...)

	// create a client for reading TD's
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
		if readtd {
			hostPort := fmt.Sprintf("%s:%d", entry.Addr, entry.Port)
			cl := tlsclient.NewTLSClient(hostPort, nil, nil, 0)
			cl.SetAuthToken(authToken)
			tdJSON, code, err := cl.Get(entry.TD)
			var tdObj map[string]any
			err = jsoniter.Unmarshal(tdJSON, &tdObj)
			tdPretty, _ := json.MarshalIndent(tdObj, "", "    ")

			if err == nil {
				fmt.Printf("--- TD BEGIN:  %v\n", tdObj["id"])
				fmt.Println(string(tdPretty))
				fmt.Printf("--- TD END: %v\n", tdObj["id"])
			} else {
				fmt.Println("Unable to read the directory TD from 'https://%s%s' (%d): %s",
					hostPort, entry.TD, code, err.Error())
			}
		}
	}

	return nil
}
