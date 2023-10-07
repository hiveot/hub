package historycli

import (
	"context"
	"fmt"
	"sort"

	"github.com/araddon/dateparse"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"

	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/pkg/history"
	"github.com/hiveot/hub/pkg/history/capnpclient"
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

func HistoryListCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:      "lev",
		Usage:     "List history of thing events",
		ArgsUsage: "<pubID> <thingID>",
		Category:  "history",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 2 {
				return fmt.Errorf("publisherID and thingID expected")
			}
			err := HandleListEvents(ctx, *runFolder, cCtx.Args().First(), cCtx.Args().Get(1), 30)
			return err
		},
	}
}

func HistoryLatestCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:      "lla",
		Usage:     "List latest values of a thing",
		ArgsUsage: "<pubID> <thingID>",
		Category:  "history",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 2 {
				return fmt.Errorf("publisherID and thingID expected")
			}
			err := HandleListLatestEvents(ctx, *runFolder, cCtx.Args().First(), cCtx.Args().Get(1))
			return err
		},
	}
}
func HistoryRetainCommand(ctx context.Context, runFolder *string) *cli.Command {
	return &cli.Command{
		Name:  "shre",
		Usage: "Show history retained events",
		//ArgsUsage: "(no args)",
		Category: "history",
		Action: func(cCtx *cli.Context) error {
			if cCtx.NArg() != 0 {
				return fmt.Errorf("no arguments expected")
			}
			err := HandleListRetainedEvents(ctx, *runFolder)
			return err
		},
	}
}

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
func HandleListEvents(ctx context.Context, runFolder string, publisherID, thingID string, limit int) error {
	var hist history.IHistoryService
	var rd history.IReadHistory

	capClient, err := hubclient.ConnectWithCapnpUDS(history.ServiceName, runFolder)
	if err == nil {
		hist = capnpclient.NewHistoryCapnpClient(capClient)
		rd, err = hist.CapReadHistory(ctx, "hubcli")
	}
	if err != nil {
		return err
	}
	eventName := ""
	cursor := rd.GetEventHistory(ctx, publisherID, thingID, eventName)
	fmt.Println("PublisherID    ThingID            Timestamp                    Event           Value (truncated)")
	fmt.Println("-----------    -------            ---------                    -----           ---------------- ")
	count := 0
	for tv, valid := cursor.Last(); valid && count < limit; tv, valid = cursor.Prev() {
		count++
		utime, err := dateparse.ParseAny(tv.Created)

		if err != nil {
			logrus.Infof("Parsing time failed '%s': %s", tv.Created, err)
		}

		fmt.Printf("%-14s %-18s %-28s %-15s %-30s\n",
			tv.PublisherID,
			tv.ThingID,
			utime.Format("02 Jan 2006 15:04:05 MST"),
			tv.ID,
			tv.Data,
		)
	}
	rd.Release()
	return err
}

// HandleListRetainedEvents lists the events that are retained
func HandleListRetainedEvents(ctx context.Context, runFolder string) error {

	var hist history.IHistoryService
	var mngRet history.IManageRetention

	capClient, err := hubclient.ConnectWithCapnpUDS(history.ServiceName, runFolder)
	if err == nil {
		hist = capnpclient.NewHistoryCapnpClient(capClient)
		mngRet, err = hist.CapManageRetention(ctx, "hubcli")
	}
	if err != nil {
		return err
	}
	evList, _ := mngRet.GetEvents(ctx)
	sort.Slice(evList, func(i, j int) bool {
		return evList[i].Name < evList[j].Name
	})

	fmt.Printf("Events (%2d)      days     publishers                     Things                         Excluded\n", len(evList))
	fmt.Println("----------       ----     ----------                     ------                         -------- ")
	for _, evRet := range evList {

		fmt.Printf("%-16.16s %-8d %-30.30s %-30.30s %-30.30s\n",
			evRet.Name,
			evRet.RetentionDays,
			fmt.Sprintf("%s", evRet.Publishers),
			fmt.Sprintf("%s", evRet.Things),
			fmt.Sprintf("%s", evRet.Exclude),
		)
	}
	mngRet.Release()
	return err
}

func HandleListLatestEvents(
	ctx context.Context, runFolder string, publisherID, thingID string) error {
	var hist history.IHistoryService
	var readHist history.IReadHistory

	capClient, err := hubclient.ConnectWithCapnpUDS(history.ServiceName, runFolder)
	if err == nil {
		hist = capnpclient.NewHistoryCapnpClient(capClient)
		readHist, err = hist.CapReadHistory(ctx, "hubcli")
	}
	if err != nil {
		return err
	}
	props := readHist.GetProperties(ctx, publisherID, thingID, nil)

	fmt.Println("Event ID         Publisher       Thing                Created                     Value")
	fmt.Println("----------         ---------       -----                -------                     -----")
	for _, prop := range props {
		utime, _ := dateparse.ParseAny(prop.Created)

		fmt.Printf("%-18.18s %-15.15s %-20s %-27s %s\n",
			prop.ID,
			prop.PublisherID,
			prop.ThingID,
			//utime.Format("02 Jan 2006 15:04:05 -0700"),
			utime.Format("02 Jan 2006 15:04:05 MST"),
			prop.Data,
		)
	}
	readHist.Release()
	return err
}
