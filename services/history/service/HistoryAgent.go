package service

import (
	"fmt"

	"github.com/hiveot/hivehub/api/go/vocab"
	"github.com/hiveot/hivehub/lib/hubagent"
	"github.com/hiveot/hivehub/services/history/historyapi"
	"github.com/hiveot/hivekitgo/messaging"
	"github.com/hiveot/hivekitgo/wot"
)

// StartHistoryAgent returns a new instance of the agent for the history services.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the history service whose capabilities to expose
//	ag is the optional connected agent connected to the server protocol
func StartHistoryAgent(svc *HistoryService, ag *messaging.Agent) {

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
	rah := hubagent.NewAgentHandler(historyapi.ReadHistoryServiceID, readHistoryMethods)
	mah := hubagent.NewAgentHandler(historyapi.ManageHistoryServiceID, manageHistoryMethods)

	// receive subscribed updates for events and properties
	ag.Consumer.SetNotificationHandler(func(notif *messaging.NotificationMessage) {
		if notif.Operation == wot.OpSubscribeEvent {
			_ = svc.addHistory.AddMessage(notif)
		} else if notif.Operation == wot.OpObserveProperty {
			_ = svc.addHistory.AddMessage(notif)
		}
		//ignore the rest
		return
	})

	// handle service requests
	ag.SetRequestHandler(func(req *messaging.RequestMessage, c messaging.IConnection) *messaging.ResponseMessage {
		if req.Operation == vocab.OpInvokeAction {
			if req.ThingID == historyapi.ReadHistoryServiceID {
				return rah.HandleRequest(req, c)
			} else if req.ThingID == historyapi.ManageHistoryServiceID {
				return mah.HandleRequest(req, c)
			}
		}
		return req.CreateResponse(nil, fmt.Errorf("Unhandled message"))
	})

	// TODO: publish the TD
}
