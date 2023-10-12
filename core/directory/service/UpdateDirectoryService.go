package service

import (
	"encoding/json"
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/lib/buckets"
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
	// The client that is updating the directory
	clientID string
	// bucket that holds the TD documents
	bucket buckets.IBucket
}

func (svc *UpdateDirectoryService) RemoveTD(senderID string, args directory.RemoveTDArgs) error {
	slog.Info("RemoveTD",
		slog.String("senderID", senderID),
		slog.String("agentID", args.AgentID),
		slog.String("thingID", args.ThingID))

	thingAddr := args.AgentID + "/" + args.ThingID
	err := svc.bucket.Delete(thingAddr)
	return err
}

func (svc *UpdateDirectoryService) UpdateTD(senderID string, args directory.UpdateTDArgs) error {
	slog.Info("UpdateTD",
		slog.String("senderID", senderID),
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

func (svc *UpdateDirectoryService) Release() {
	_ = svc.bucket.Close()
}

// NewUpdateDirectoryService returns the capability to update the directory
// bucket with the TD documents. Will be closed when done.
func NewUpdateDirectoryService(clientID string, bucket buckets.IBucket) (
	*UpdateDirectoryService, map[string]interface{}) {

	svc := &UpdateDirectoryService{
		clientID: clientID,
		bucket:   bucket,
	}
	capabilityMap := map[string]interface{}{
		directory.UpdateTDMethod: svc.UpdateTD,
		directory.RemoveTDMethod: svc.RemoveTD,
	}
	return svc, capabilityMap
}
