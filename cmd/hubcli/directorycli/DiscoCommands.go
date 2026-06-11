package directorycli

import (
	"encoding/json"
	"fmt"
	"time"

	discoverypkg "github.com/hiveot/hivekit/go/modules/transport/discovery/pkg"
	tlsclientpkg "github.com/hiveot/hivekit/go/modules/transport/tlsclient/pkg"
	jsoniter "github.com/json-iterator/go"
	"github.com/urfave/cli/v2"
)

// DiscoListCommand lists discovered Thing and Directory servers
//
// authToken used to read the TD
func DiscoListCommand(clientID string, authToken *string) *cli.Command {
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
			err := HandleDiscover(clientID, *authToken, readtd)
			return err
		},
	}
}

// HandleDiscover prints a list of discovered Things and Directories
func HandleDiscover(clientID string, authToken string, readtd bool) error {
	discoClient := discoverypkg.NewDiscoveryClient()
	allRecords, err := discoClient.DiscoverDirectories("", time.Second*2, false, nil)
	if err != nil {
		fmt.Println("Discovery failed: ", err.Error())
		return err
	}
	//hiveotRecords := discovery.DiscoverTDD("", "hiveot", time.Second*2, false)
	//allRecords := append(hiveotRecords, wotRecords...)

	// create a client for reading TD's
	fmt.Println("Address                     Port  Instance       Type      Schema   TD path")
	fmt.Println("-------                    -----  --------       ----      ------   -------")
	for _, entry := range allRecords {
		fmt.Printf("%-25s %6d  %-11s %10s   %-8s %s\n",
			entry.Addr,
			entry.Port,
			entry.Instance,
			entry.Type,
			entry.Schema,
			entry.TD,
		)
		if readtd {
			var tdObj map[string]any

			hostPort := fmt.Sprintf("%s:%d", entry.Addr, entry.Port)
			cl := tlsclientpkg.NewTLSClient(hostPort, nil, 0)
			err := cl.AuthenticateWithToken(clientID, authToken)
			if err != nil {
				break
			}
			tdJSON, code, err2 := cl.Get(entry.TD)
			err = err2
			_ = code
			if err == nil {
				err = jsoniter.Unmarshal(tdJSON, &tdObj)
			}

			tdPretty, _ := json.MarshalIndent(tdObj, "", "    ")

			if err == nil {
				fmt.Printf("--- TD BEGIN:  %v\n", tdObj["id"])
				fmt.Println(string(tdPretty))
				fmt.Printf("--- TD END: %v\n", tdObj["id"])
			} else {
				fmt.Printf("Unable to read the directory TD from 'https://%s%s' (%d): %s\n",
					hostPort, entry.TD, code, err.Error())
			}
		}
	}

	return nil
}
