package directorycli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hiveot/hivekit/go/api"
	"github.com/hiveot/hivekit/go/cells/discovery"
	discovery_client "github.com/hiveot/hivekit/go/cells/discovery/client"
	"github.com/hiveot/hivekit/go/cells/transport/tlsclient"
	jsoniter "github.com/json-iterator/go"
	"github.com/urfave/cli/v3"
)

// DiscoListCommand lists discovered Thing and Directory servers
//
// authToken used to read the TD
func DiscoListCommand(env **api.HiveEnvironment) *cli.Command {
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
		}, Action: func(ctx context.Context, cmd *cli.Command) error {
			if cmd.NArg() != 0 {
				return fmt.Errorf("no arguments expected")
			}
			err := HandleDiscover(*env, readtd)
			return err
		},
	}
}

// HandleDiscover prints a list of discovered Things and Directories
func HandleDiscover(env *api.HiveEnvironment, readtd bool) error {
	discoClient, err := discovery_client.NewDiscoveryClient(env, true)
	authToken, _ := env.GetAuthToken()
	if err != nil {
		return err
	}
	allRecords, err := discoClient.DiscoverThings("", discovery.DISCO_TYPE_THING, false, time.Second*2, nil)
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
			cl := tlsclient.NewTLSClient(hostPort, nil)
			err := cl.SetAuthToken(env.ClientID, authToken)
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
