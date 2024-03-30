package service

import (
	"encoding/json"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/things"
	"log/slog"
)

// UpdateDirectoryService is a provides the capability to update the directory
// This implements the IUpdateDirectory API
//
//	Bucket keys are made of gatewayID+"/"+thingID
//	Bucket values are ThingValue objects
type UpdateDirectoryService struct {
	// bucket that holds the TD documents
	bucket buckets.IBucket
}

// CreateUpdateDirTD a new Thing TD document describing the update directory capability
func (svc *UpdateDirectoryService) CreateUpdateDirTD() *things.TD {
	title := "Thing Directory Updater"
	deviceType := vocab.ThingServiceDirectory
	td := things.NewTD(directoryapi.UpdateDirectoryCap, title, deviceType)
	// TODO: add properties
	return td
}

func (svc *UpdateDirectoryService) RemoveTD(ctx hubclient.ServiceContext, args directoryapi.RemoveTDArgs) error {
	slog.Info("RemoveTD",
		slog.String("senderID", ctx.SenderID),
		slog.String("agentID", args.AgentID),
		slog.String("thingID", args.ThingID))

	thingAddr := args.AgentID + "/" + args.ThingID
	err := svc.bucket.Delete(thingAddr)
	return err
}

func (svc *UpdateDirectoryService) UpdateTD(ctx hubclient.ServiceContext, args directoryapi.UpdateTDArgs) error {
	slog.Info("UpdateTD",
		slog.String("senderID", ctx.SenderID),
		slog.String("agentID", args.AgentID),
		slog.String("thingID", args.ThingID))

	// store the TD ThingValue
	thingValue := things.NewThingValue(
		transports.MessageTypeEvent, args.AgentID, args.ThingID, transports.EventNameTD, args.TDDoc, ctx.SenderID)
	bucketData, _ := json.Marshal(thingValue)
	thingAddr := args.AgentID + "/" + args.ThingID
	err := svc.bucket.Set(thingAddr, bucketData)
	return err
}

// Stop the update directory capability
// This unsubscribes from requests.
func (svc *UpdateDirectoryService) Stop() {
}

// StartUpdateDirectoryService starts the capability to update the directory.
// Invoke Stop() when done to unsubscribe from requests.
//
//	hc with the message bus connection
//	thingBucket is the open bucket used to store TDs
func StartUpdateDirectoryService(hc *hubclient.HubClient, bucket buckets.IBucket) *UpdateDirectoryService {

	svc := &UpdateDirectoryService{
		bucket: bucket,
	}
	capMethods := map[string]interface{}{
		directoryapi.UpdateTDMethod: svc.UpdateTD,
		directoryapi.RemoveTDMethod: svc.RemoveTD,
	}
	hc.SetRPCCapability(directoryapi.UpdateDirectoryCap, capMethods)

	return svc
}
