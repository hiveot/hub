package pubsubcli

import (
	"fmt"
	"github.com/hiveot/hub/api/go/digitwin"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/wot/td"
	"github.com/hiveot/hub/wot/transports"
	utils2 "github.com/hiveot/hub/wot/transports/utils"
	jsoniter "github.com/json-iterator/go"
	"time"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v2"
)

// SubTDCommand shows TD publications
func SubTDCommand(hc *transports.IClientConnection) *cli.Command {
	return &cli.Command{
		Name:     "subtd",
		Usage:    "SubscribeEvent to TD publications",
		Category: "pubsub",
		Action: func(cCtx *cli.Context) error {
			err := HandleSubTD(*hc)
			return err
		},
	}
}

func SubEventsCommand(hc *transports.IClientConnection) *cli.Command {
	return &cli.Command{
		Name:      "subev",
		Usage:     "SubscribeEvent to Thing events",
		ArgsUsage: "[<thingID> [<key>]]",
		Category:  "pubsub",
		Action: func(cCtx *cli.Context) error {
			thingID := ""
			key := ""
			if cCtx.NArg() > 0 {
				thingID = cCtx.Args().Get(0)
			}
			if cCtx.NArg() > 1 {
				key = cCtx.Args().Get(1)
			}
			if cCtx.NArg() > 2 {
				return fmt.Errorf("Unexpected arguments")
			}

			err := HandleSubEvents(*hc, thingID, key)
			return err
		},
	}
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(hc transports.IClientConnection) error {

	err := hc.Subscribe(digitwin.DirectoryDThingID, digitwin.DirectoryEventThingUpdated)
	if err != nil {
		return err
	}
	hc.SetNotificationHandler(func(msg *transports.ThingMessage) {
		// only look for TD events, ignore directed events
		if msg.Name != digitwin.DirectoryEventThingUpdated {
			return
		}

		var td td.TD
		//fmt.Printf("%s\n", event.ValueJSON)
		err := utils2.DecodeAsObject(msg.Data, &td)

		if err == nil {
			modifiedTime, _ := dateparse.ParseAny(td.Modified) // can be in any TZ
			timeStr := utils.FormatMSE(modifiedTime.In(time.Local).UnixMilli(), false)
			fmt.Printf("%-20.20s %-35.35s %-30.30s %-30.30s %-30.30s\n",
				msg.SenderID, msg.ThingID, td.Title, td.AtType, timeStr)
		}
		return
	})
	fmt.Printf("Sender ID            Thing ID                            Title                          @type                          Updated                       \n")
	fmt.Printf("-------------------  ----------------------------------  -----------------------------  -----------------------------  ------------------------------\n")

	time.Sleep(time.Hour * 24)
	return nil
}

// HandleSubEvents subscribes and prints events
func HandleSubEvents(hc transports.IClientConnection, thingID string, name string) error {
	fmt.Printf("Subscribing to  thingID: '%s', name: '%s'\n\n", thingID, name)

	fmt.Printf("Time             Agent ID        Thing ID                       Event Name                     Value\n")
	fmt.Printf("---------------  --------------- -----------------------------  -----------------------------  ---------\n")

	err := hc.Subscribe(thingID, name)
	hc.SetNotificationHandler(func(msg *transports.ThingMessage) {
		createdTime, _ := dateparse.ParseAny(msg.Created)
		timeStr := createdTime.Format("15:04:05.000")

		valueStr := msg.DataAsText()

		//if msg.Name == vocab.EventNameProperties {
		//	var props map[string]interface{}
		//	_ = utils.DecodeAsObject(msg.Data, &props)
		//
		//	valueStr = fmt.Sprintf("%d properties", len(props))
		//
		//	// if its only a single property then show the value
		//	if len(props) == 1 {
		//		for key, val := range props {
		//			valueStr = fmt.Sprintf("{%s=%v}", key, val)
		//		}
		//	}
		//}
		if msg.ThingID == digitwin.DirectoryDThingID && msg.Name == digitwin.DirectoryEventThingUpdated {
			var td td.TD
			tdJSON := msg.DataAsText()
			jsoniter.UnmarshalFromString(tdJSON, &td)
			valueStr = fmt.Sprintf("{title:%s, type:%s, nrProps=%d, nrEvents=%d, nrActions=%d}",
				td.Title, td.AtType, len(td.Properties), len(td.Events), len(td.Actions))
		}

		fmt.Printf("%-16.16s %-15.15s %-30.30s %-30.30s %-40.40s\n",
			timeStr, msg.SenderID, msg.ThingID, msg.Name, valueStr)
		return
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}
