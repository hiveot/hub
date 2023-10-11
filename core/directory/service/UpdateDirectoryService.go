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

func (svc *UpdateDirectoryService) RemoveTD(agentID, thingID string) error {
	slog.Info("RemoveTD", "clientID", svc.clientID,
		"agentID", agentID, "thingID", thingID)
	thingAddr := agentID + "/" + thingID
	err := svc.bucket.Delete(thingAddr)
	return err
}

func (svc *UpdateDirectoryService) UpdateTD(agentID, thingID string, td []byte) error {
	//logrus.Infof("clientID=%s, thingID=%s", svc.clientID, thingID)

	thingValue := thing.NewThingValue(agentID, thingID, vocab.EventNameTD, td)
	bucketData, _ := json.Marshal(thingValue)
	thingAddr := agentID + "/" + thingID
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
