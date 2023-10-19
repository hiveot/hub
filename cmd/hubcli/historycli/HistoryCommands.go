package historycli

import (
	"fmt"
	"github.com/hiveot/hub/core/history/historyclient"
	"time"

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
	return &cli.Command{
		Name:      "lev",
		Usage:     "List history of thing events",
		ArgsUsage: "<agentID> <thingID>",
		Category:  "history",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 2 {
				return fmt.Errorf("agentID and thingID expected")
			}
			err := HandleListEvents(*hc, cCtx.Args().First(), cCtx.Args().Get(1), 30)
			return err
		},
	}
}

func HistoryLatestCommand(hc *hubclient.IHubClient) *cli.Command {
	return &cli.Command{
		Name:      "lla",
		Usage:     "List latest values of a thing",
		ArgsUsage: "<pubID> <thingID>",
		Category:  "history",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 2 {
				return fmt.Errorf("publisherID and thingID expected")
			}
			err := HandleListLatestEvents(*hc, cCtx.Args().First(), cCtx.Args().Get(1))
			return err
		},
	}
}

//func HistoryRetainCommand(hc *hubclient.IHubClient) *cli.Command {
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
func HandleListEvents(hc hubclient.IHubClient, publisherID, thingID string, limit int) error {
	rd := historyclient.NewReadHistoryClient(hc)
	cursor, _, err := rd.GetCursor("", "", "")
	fmt.Println("AgentID        ThingID            Timestamp                    Event           Value (truncated)")
	fmt.Println("-----------    -------            ---------                    -----           ---------------- ")
	count := 0
	for tv, valid, err := cursor.Last(); err == nil && valid && count < limit; tv, valid, err = cursor.Prev() {
		count++
		utime := time.UnixMilli(tv.CreatedMSec)

		fmt.Printf("%-14s %-18s %-28s %-15s %-30s\n",
			tv.AgentID,
			tv.ThingID,
			utime.Format("02 Jan 2006 15:04:05 MST"),
			tv.Name,
			tv.Data,
		)
	}
	cursor.Release()
	return err
}

//
//// HandleListRetainedEvents lists the events that are retained
//func HandleListRetainedEvents(hc hubclient.IHubClient) error {
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

func HandleListLatestEvents(
	hc hubclient.IHubClient, agentID string, thingID string) error {
	rd := historyclient.NewReadHistoryClient(hc)

	tvList, err := rd.GetLatest(agentID, thingID, nil)

	fmt.Println("Event ID         Publisher       Thing                CreatedMSec                     Value")
	fmt.Println("----------         ---------       -----                -------                     -----")
	for _, tv := range tvList {
		utime := time.UnixMilli(tv.CreatedMSec)

		fmt.Printf("%-18.18s %-15.15s %-20s %-27s %s\n",
			tv.Name,
			tv.AgentID,
			tv.ThingID,
			//utime.Format("02 Jan 2006 15:04:05 -0700"),
			utime.Format("02 Jan 2006 15:04:05 MST"),
			tv.Data,
		)
	}
	return err
}
