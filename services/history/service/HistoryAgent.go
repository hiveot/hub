package service

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubagent"
	"github.com/hiveot/hub/services/history/historyapi"
	"github.com/hiveot/hub/transports"
)

// StartHistoryAgent returns a new instance of the agent for the history services.
// This uses the given connected transport for publishing events and subscribing to actions.
// The transport must be closed by the caller after use.
// If the transport is nil then use the HandleMessage method directly to pass methods to the agent,
// for example when testing.
//
//	svc is the history service whose capabilities to expose
//	hc is the optional message client connected to the server protocol
func StartHistoryAgent(svc *HistoryService, hc transports.IAgentConnection) {

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
	hc.SetNotificationHandler(func(msg transports.NotificationMessage) {
		if msg.Operation == vocab.HTOpPublishEvent {
			_ = svc.addHistory.AddEvent(&msg)
		} else if msg.Operation == vocab.HTOpUpdateProperty {
			_ = svc.addHistory.AddProperty(&msg)
		} else {
			//ignore the rest
		}
	})

	// handle service requests
	hc.SetRequestHandler(func(req transports.RequestMessage) transports.ResponseMessage {
		if req.Operation == vocab.OpInvokeAction {
			if req.ThingID == historyapi.ReadHistoryServiceID {
				return rah.HandleRequest(req)
			} else if req.ThingID == historyapi.ManageHistoryServiceID {
				return mah.HandleRequest(req)
			}
		}
		return req.CreateResponse(nil, fmt.Errorf("Unhandled message"))
	})

	// TODO: publish the TD
}
