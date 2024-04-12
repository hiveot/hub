package rpc

import (
	"encoding/json"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/directory/service"
	"log/slog"
)

// DirectoryRPC contains the message based interface to the directory service
// This implements a HandleMessage method that supports the message format from the API.
type DirectoryRPC struct {
	svc *service.DirectoryService
}

// HandleMessage an event or action message for the directory service
func (rpc *DirectoryRPC) HandleMessage(msg *things.ThingMessage) ([]byte, error) {
	if msg.MessageType == vocab.MessageTypeEvent && msg.Key == vocab.EventTypeTD {
		return nil, rpc.HandleTDEvent(msg)
	} else if msg.MessageType == vocab.MessageTypeAction {
		// all rpc calls
		if msg.Key == api.DirectoryReadTDDsMethod {
			return rpc.HandleReadTDDs(msg)
		} else if msg.Key == api.DirectoryReadTDDMethod {
			return rpc.HandleReadTDD(msg)
		} else if msg.Key == api.DirectoryRemoveTDDMethod {
			return rpc.HandleRemoveTDD(msg)
		}
	}
	return nil, nil
}

// HandleReadTDDs handles an action request for a list of TD documents
func (rpc *DirectoryRPC) HandleReadTDDs(msg *things.ThingMessage) ([]byte, error) {
	var args api.DirectoryReadTDDsArgs
	var tdds []*things.TD

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		tdds, err = rpc.svc.ReadTDDs(args.Offset, args.Limit)
	}
	if err != nil {
		slog.Warn("HandleReadTDDs failed", "err", err)
	}
	resp := api.DirectoryReadTDDsResp{
		TDDs: tdds,
	}
	respJson, err := json.Marshal(resp)
	return respJson, err
}

// HandleReadTDD handles an action request for a sing TD document
func (rpc *DirectoryRPC) HandleReadTDD(msg *things.ThingMessage) ([]byte, error) {
	var args api.DirectoryReadTDDArgs
	var tdd *things.TD

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		tdd, err = rpc.svc.ReadTDD(args.ThingID)
	}
	if err != nil {
		slog.Warn("HandleReadTDD failed", "err", err)
	}
	resp := api.DirectoryReadTDDResp{
		TDD: tdd,
	}
	respJson, err := json.Marshal(resp)
	return respJson, err
}

// HandleRemoveTDD handles an action request for removing a TD document
func (rpc *DirectoryRPC) HandleRemoveTDD(msg *things.ThingMessage) ([]byte, error) {
	var args api.DirectoryRemoveTDDArgs

	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = rpc.svc.RemoveTDD(msg.SenderID, args.ThingID)
	}
	if err != nil {
		slog.Warn("HandleRemoveTDD failed", "err", err)
	}
	return nil, err
}

// HandleTDEvent handles an event containing a TD document
func (rpc *DirectoryRPC) HandleTDEvent(msg *things.ThingMessage) error {
	var args api.DirectoryTDEventArgs
	err := json.Unmarshal(msg.Data, &args)
	if err == nil {
		err = rpc.svc.UpdateTDD(msg.SenderID, msg.ThingID, args)
	}
	if err != nil {
		slog.Warn("HandleEvent: TD update failed", "err", err)
	}
	return err
}

func NewDirectoryRPC(svc *service.DirectoryService) *DirectoryRPC {
	rpc := DirectoryRPC{svc: svc}
	return &rpc
}
