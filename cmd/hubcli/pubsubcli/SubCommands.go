package pubsubcli

import (
	"fmt"
	"github.com/hiveot/hub/lib/utils"
	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/consumer"
	"github.com/hiveot/hub/messaging/tputils"
	digitwin "github.com/hiveot/hub/runtime/digitwin/api"
	"github.com/hiveot/hub/wot/td"
	jsoniter "github.com/json-iterator/go"
	"time"

	"github.com/araddon/dateparse"
	"github.com/urfave/cli/v2"
)

// SubTDCommand shows TD publications
func SubTDCommand(hc **consumer.Consumer) *cli.Command {
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

func SubEventsCommand(hc **consumer.Consumer) *cli.Command {
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
				return fmt.Errorf("unexpected arguments")
			}

			err := HandleSubEvents(*hc, thingID, key)
			return err
		},
	}
}

// HandleSubTD subscribes and prints TD publications
func HandleSubTD(hc *consumer.Consumer) error {

	err := hc.Subscribe(digitwin.ThingDirectoryDThingID, digitwin.ThingDirectoryEventThingUpdated)
	if err != nil {
		return err
	}
	hc.SetResponseHandler(func(msg *messaging.ResponseMessage) error {
		// only look for TD events, ignore directed events
		if msg.Name != digitwin.ThingDirectoryEventThingUpdated {
			return nil
		}

		var tdi td.TD
		//fmt.Printf("%s\n", event.ValueJSON)
		err := tputils.DecodeAsObject(msg.Output, &tdi)

		if err == nil {
			modifiedTime, _ := dateparse.ParseAny(tdi.Modified) // can be in any TZ
			timeStr := utils.FormatMSE(modifiedTime.In(time.Local).UnixMilli(), false)
			fmt.Printf("%-20.20s %-35.35s %-30.30s %-30.30s %-30.30s\n",
				"", msg.ThingID, tdi.Title, tdi.AtType, timeStr)
		}
		return nil
	})
	fmt.Printf("Sender ID            Thing ID                            Title                          @type                          Updated                       \n")
	fmt.Printf("-------------------  ----------------------------------  -----------------------------  -----------------------------  ------------------------------\n")

	time.Sleep(time.Hour * 24)
	return nil
}

// HandleSubEvents subscribes and prints events
func HandleSubEvents(hc *consumer.Consumer, thingID string, name string) error {
	fmt.Printf("Subscribing to  thingID: '%s', name: '%s'\n\n", thingID, name)

	fmt.Printf("Time             Agent ID        Thing ID                       Event Name                     Value\n")
	fmt.Printf("---------------  --------------- -----------------------------  -----------------------------  ---------\n")

	err := hc.Subscribe(thingID, name)
	hc.SetResponseHandler(func(msg *messaging.ResponseMessage) error {
		createdTime, _ := dateparse.ParseAny(msg.Updated)
		timeStr := createdTime.Format("15:04:05.000")

		valueStr := msg.ToString(0)

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
		if msg.ThingID == digitwin.ThingDirectoryDThingID &&
			msg.Name == digitwin.ThingDirectoryEventThingUpdated {
			var tdi td.TD
			tdJSON := msg.ToString(0)
			_ = jsoniter.UnmarshalFromString(tdJSON, &tdi)
			valueStr = fmt.Sprintf("{title:%s, type:%s, nrProps=%d, nrEvents=%d, nrActions=%d}",
				tdi.Title, tdi.AtType, len(tdi.Properties), len(tdi.Events), len(tdi.Actions))
		}

		fmt.Printf("%-16.16s %-15.15s %-30.30s %-30.30s %-40.40s\n",
			timeStr, "", msg.ThingID, msg.Name, valueStr)
		return nil
	})
	if err != nil {
		return err
	}
	time.Sleep(time.Hour * 24)
	return nil
}
