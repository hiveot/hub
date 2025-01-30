package historycli

import (
	"fmt"
	"github.com/hiveot/hub/services/history/historyclient"
	"github.com/hiveot/hub/transports/messaging"
	"github.com/urfave/cli/v2"
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

func HistoryListCommand(hc **messaging.Consumer) *cli.Command {
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

//func HistoryLatestCommand(hc *hubclient.IConsumer) *cli.Command {
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
func HandleListEvents(hc *messaging.Consumer, dThingID string, name string, limit int) error {
	// FIXME: hc has a bootstrap algo to read the needed TD
	//histTD := hc.GetTD(historyapi.ReadHistoryServiceID)
	//f := histTD.GetForm(wot.OpInvokeAction)
	rd := historyclient.NewReadHistoryClient(hc)

	cursor, releaseFn, err := rd.GetCursor(dThingID, name)
	defer releaseFn()
	if err != nil {
		return err
	}
	fmt.Println("ThingID                        Timestamp                      Event                Value (truncated)")
	fmt.Println("-----------                    ---------                      -----                ---------------- ")
	count := 0
	for msg, valid, err := cursor.Last(); err == nil && valid && count < limit; msg, valid, err = cursor.Prev() {
		count++
		value := msg.ToString(30)
		// show number of properties
		//if msg.Name == vocab.EventNameProperties {
		//	props := make(map[string]interface{})
		//	_ = utils.DecodeAsObject(msg.Data, &props)
		//	value = fmt.Sprintf("(%d properties)", len(props))
		//}
		// FIXME: reformat timestmp
		updated := msg.Updated
		fmt.Printf("%-30s %-30s %-20.20s %-30s\n",
			msg.ThingID,
			updated,
			msg.Name,
			value,
		)
	}
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
//	hc hubclient.IConsumer, agentID string, thingID string) error {
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
