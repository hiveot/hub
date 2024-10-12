package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/transports"
	"github.com/hiveot/hub/services/history/historyapi"
)

// StartHistoryAgent returns a new instance of the agent for the history services.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the history service whose capabilities to expose
//	hc is the optional message client connected to the server protocol
func StartHistoryAgent(
	svc *HistoryService, hc hubclient.IConsumerClient) {

	// TODO: load latest retention rules from state store
	manageHistoryMethods := map[string]interface{}{
		historyapi.GetRetentionRuleMethod:  svc.manageHistSvc.GetRetentionRule,
		historyapi.GetRetentionRulesMethod: svc.manageHistSvc.GetRetentionRules,
		historyapi.SetRetentionRulesMethod: svc.manageHistSvc.SetRetentionRules,
	}
	readHistoryMethods := map[string]interface{}{
		historyapi.CursorFirstMethod:   svc.readHistSvc.First,
		historyapi.CursorLastMethod:    svc.readHistSvc.Last,
		historyapi.CursorNextMethod:    svc.readHistSvc.Next,
		historyapi.CursorNextNMethod:   svc.readHistSvc.NextN,
		historyapi.CursorPrevMethod:    svc.readHistSvc.Prev,
		historyapi.CursorPrevNMethod:   svc.readHistSvc.PrevN,
		historyapi.CursorReleaseMethod: svc.readHistSvc.Release,
		historyapi.CursorSeekMethod:    svc.readHistSvc.Seek,
		historyapi.GetCursorMethod:     svc.readHistSvc.GetCursor,
		historyapi.ReadHistoryMethod:   svc.readHistSvc.ReadHistory,
	}
	rah := transports.NewAgentHandler(historyapi.ReadHistoryServiceID, readHistoryMethods)
	mah := transports.NewAgentHandler(historyapi.ManageHistoryServiceID, manageHistoryMethods)

	// receive messages for events and agent requests
	hc.SetMessageHandler(func(msg *hubclient.ThingMessage) (stat hubclient.DeliveryStatus) {
		if msg.MessageType == vocab.MessageTypeAction || msg.MessageType == vocab.MessageTypeProperty {
			if msg.ThingID == historyapi.ReadHistoryServiceID {
				return rah.HandleMessage(msg)
			} else if msg.ThingID == historyapi.ManageHistoryServiceID {
				return mah.HandleMessage(msg)
			}
		} else if msg.MessageType == vocab.MessageTypeEvent {
			err := svc.addHistory.AddEvent(msg)
			return *stat.Completed(msg, nil, err)
		}
		stat.Failed(msg, fmt.Errorf("Unhandled message"))
		return stat
	})
}
