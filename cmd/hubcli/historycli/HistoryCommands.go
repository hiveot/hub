package historycli

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
)

//func HistoryInfoCommand(ctx context.Context, runFolder *string) *cli.Command {
//	return &cli.Command{
//		Name:     "hsi",
//		Usage:    "Show history store info",
//		Category: "history",
//		//ArgsUsage: "(no args)",
//		Action: func(cCtx *cli.Context) error {
//			if cCtx.NArg() != 0 {
//				return fmt.Errorf("no arguments expected")
//			}
//			err := HandleHistoryInfo(ctx, *runFolder)
//			return err
//		},
//	}
//}

func HistoryListCommand(hc *hubclient.IHubClient) *cli.Command {
	limit := 100
	return &cli.Command{
		Name:      "hev",
		Usage:     "History of Thing events",
		ArgsUsage: "<thingID> [<key>]",
		Category:  "history",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "limit",
				Usage:       "Nr of events the show",
				Value:       limit,
				Destination: &limit,
			},
		},
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() < 1 {
				return fmt.Errorf("thingID expected")
			}
			key := ""
			if cCtx.NArg() == 2 {
				key = cCtx.Args().Get(1)
			}
			err := HandleListEvents(*hc, cCtx.Args().First(), key, limit)
			return err
		},
	}
}

//func HistoryLatestCommand(hc *hubclient.IHubClient) *cli.Command {
//	return &cli.Command{
//		Name:      "hla",
//		Usage:     "History latest values of a things",
//		ArgsUsage: "<pubID> <thingID>",
//		Category:  "history",
//		Action: func(cCtx *cli.Context) error {
//			if cCtx.NArg() != 2 {
//				return fmt.Errorf("publisherID and thingID expected")
//			}
//			err := HandleListLatestEvents(*hc, cCtx.Args().First(), cCtx.Args().Get(1))
//			return err
//		},
//	}
//}

//func HistoryRetainCommand(hc **hubclient.HubClient) *cli.Command {
//	return &cli.Command{
//		Name:  "shre",
//		Usage: "Show history retained events",
//		//ArgsUsage: "(no args)",
//		Category: "history",
//		Action: func(cCtx *cli.Context) error {
//			if cCtx.NArg() != 0 {
//				return fmt.Errorf("no arguments expected")
//			}
//			err := HandleListRetainedEvents(*hc)
//			return err
//		},
//	}
//}

//func HandleHistoryInfo(ctx context.Context, runFolder string) error {
//	var hist history.IHistoryService
//	var rd history.IReadHistory
//
//	capClient, err := hubclient.ConnectWithCapnpUDS(history.ServiceName, runFolder)
//	if err == nil {
//		hist = capnpclient.NewHistoryCapnpClient(capClient)
//		rd, err = hist.CapReadHistory(ctx, "hubcli", "", "")
//	}
//	if err != nil {
//		return err
//	}
//	info := rd.Info(ctx)
//
//	fmt.Println(fmt.Sprintf("ID:          %s", info.Id))
//	fmt.Println(fmt.Sprintf("Size:        %d", info.DataSize))
//	fmt.Println(fmt.Sprintf("Nr Records   %d", info.NrRecords))
//	fmt.Println(fmt.Sprintf("Engine       %s", info.Engine))
//
//	rd.Release()
//	return err
//}

// HandleListEvents lists the history content
func HandleListEvents(hc hubclient.IHubClient, dThingID string, name string, limit int) error {
	rd := historyclient.NewReadHistoryClient(hc)

	cursor, releaseFn, err := rd.GetCursor(dThingID, name)
	defer releaseFn()
	if err != nil {
		return err
	}
	fmt.Println("ThingID                        Timestamp                      Event                Value (truncated)")
	fmt.Println("-----------                    ---------                      -----                ---------------- ")
	count := 0
	for tv, valid, err := cursor.First(); err == nil && valid && count < limit; tv, valid, err = cursor.Next() {
		count++
		value := tv.DataAsText()
		// show number of properties
		if tv.Key == vocab.EventTypeProperties {
			props := make(map[string]string)
			_ = tv.Decode(&props)
			value = fmt.Sprintf("(%d properties)", len(props))
		}

		fmt.Printf("%-30s %-30s %-20.20s %-30.30s\n",
			tv.ThingID,
			utils.FormatMSE(tv.CreatedMSec, false),
			tv.Key,
			value,
		)
	}
	cursor.Release()
	return err
}

//
//// HandleListRetainedEvents lists the events that are retained
//func HandleListRetainedEvents(hc *hubclient.HubClient) error {
//
//	var hist history.IHistoryService
//	var mngRet history.IManageRetention
//
//	capClient, err := hubclient.ConnectWithCapnpUDS(history.ServiceName, runFolder)
//	if err == nil {
//		hist = capnpclient.NewHistoryCapnpClient(capClient)
//		mngRet, err = hist.CapManageRetention(ctx, "hubcli")
//	}
//	if err != nil {
//		return err
//	}
//	evList, _ := mngRet.GetEvents(ctx)
//	sort.Slice(evList, func(i, j int) bool {
//		return evList[i].Name < evList[j].Name
//	})
//
//	fmt.Printf("Events (%2d)      days     publishers                     Things                         Excluded\n", len(evList))
//	fmt.Println("----------       ----     ----------                     ------                         -------- ")
//	for _, evRet := range evList {
//
//		fmt.Printf("%-16.16s %-8d %-30.30s %-30.30s %-30.30s\n",
//			evRet.Name,
//			evRet.RetentionDays,
//			fmt.Sprintf("%s", evRet.Agents),
//			fmt.Sprintf("%s", evRet.Things),
//			fmt.Sprintf("%s", evRet.Exclude),
//		)
//	}
//	mngRet.Release()
//	return err
//}

//func HandleListLatestEvents(
//	hc hubclient.IHubClient, agentID string, thingID string) error {
//	rd := historyclient.NewReadHistoryClient(hc)
//
//	props, err := rd.GetLatest(agentID, thingID, nil)
//
//	fmt.Println("Event ID                  AgentID         ThingID              Value                            Created")
//	fmt.Println("--------                  -------         -------              -----                            -------")
//	for _, tv := range props {
//
//		fmt.Printf("%-25.25s %-15.15s %-20s %-32s %.80s\n",
//			tv.Name,
//			tv.AgentID,
//			tv.ThingID,
//			fmt.Sprintf("%.32s", tv.Data),
//			//utime.Format("02 Jan 2006 15:04:05 -0700"),
//			utils.FormatMSE(tv.CreatedMSec, false),
//		)
//	}
//	return err
//}
