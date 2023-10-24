package service

import (
	"encoding/json"
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
)

// UpdateDirectoryService is a provides the capability to update the directory
// This implements the IUpdateDirectory API
//
//	Bucket keys are made of gatewayID+"/"+thingID
//	Bucket values are ThingValue objects
type UpdateDirectoryService struct {
	// bucket that holds the TD documents
	bucket    buckets.IBucket
	updateSub hubclient.ISubscription
}

// CreateUpdateDirTD a new Thing TD document describing the update directory capability
func (svc *UpdateDirectoryService) CreateUpdateDirTD() *thing.TD {
	title := "Thing Directory Updater"
	deviceType := vocab.DeviceTypeService
	td := thing.NewTD(directoryapi.UpdateDirectoryCap, title, deviceType)
	// TODO: add properties
	return td
}

func (svc *UpdateDirectoryService) RemoveTD(ctx hubclient.ServiceContext, args directoryapi.RemoveTDArgs) error {
	slog.Info("RemoveTD",
		slog.String("senderID", ctx.ClientID),
		slog.String("agentID", args.AgentID),
		slog.String("thingID", args.ThingID))

	thingAddr := args.AgentID + "/" + args.ThingID
	err := svc.bucket.Delete(thingAddr)
	return err
}

func (svc *UpdateDirectoryService) UpdateTD(ctx hubclient.ServiceContext, args directoryapi.UpdateTDArgs) error {
	slog.Info("UpdateTD",
		slog.String("senderID", ctx.ClientID),
		slog.String("agentID", args.AgentID),
		slog.String("thingID", args.ThingID))

	// store the TD ThingValue
	thingValue := thing.NewThingValue(
		args.AgentID, args.ThingID, vocab.EventNameTD, args.TDDoc)
	bucketData, _ := json.Marshal(thingValue)
	thingAddr := args.AgentID + "/" + args.ThingID
	err := svc.bucket.Set(thingAddr, bucketData)
	return err
}

// Stop the update directory capability
// This unsubscribes from requests.
func (svc *UpdateDirectoryService) Stop() {
	svc.updateSub.Unsubscribe()
}

// StartUpdateDirectoryService starts the capability to update the directory.
// Invoke Stop() when done to unsubscribe from requests.
//
//	hc with the message bus connection
//	thingBucket is the open bucket used to store TDs
func StartUpdateDirectoryService(hc hubclient.IHubClient, bucket buckets.IBucket) (
	svc *UpdateDirectoryService, err error) {

	svc = &UpdateDirectoryService{
		bucket: bucket,
	}
	capMethods := map[string]interface{}{
		directoryapi.UpdateTDMethod: svc.UpdateTD,
		directoryapi.RemoveTDMethod: svc.RemoveTD,
	}
	svc.updateSub, err = hubclient.SubRPCCapability(
		hc, directoryapi.UpdateDirectoryCap, capMethods)

	return svc, err
}
