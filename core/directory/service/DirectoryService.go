package service

import (
	"encoding/json"
	"github.com/hiveot/hub/core/directory"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
)

const TDBucketName = "td"

// DirectoryService is a wrapper around the internal bucket store
// This implements the IDirectory interface
type DirectoryService struct {
	hc           hubclient.IHubClient
	store        buckets.IBucketStore
	serviceID    string // thingID of the service instance
	tdBucketName string

	// td event subscription
	tdSub hubclient.ISubscription

	// capabilities and subscriptions
	readDirSvc   *ReadDirectoryService
	readSub      hubclient.ISubscription
	updateDirSvc *UpdateDirectoryService
	updateSub    hubclient.ISubscription
}

// Create a new Thing TD document describing the read directory capability
func (svc *DirectoryService) createReadDirTD() *thing.TD {
	title := "Thing Directory Reader"
	deviceType := vocab.DeviceTypeService
	td := thing.NewTD(directory.ReadDirectoryCapability, title, deviceType)
	// TODO: add properties
	return td
}

// Create a new Thing TD document describing the update directory capability
func (svc *DirectoryService) createUpdateDirTD() *thing.TD {
	title := "Thing Directory Updater"
	deviceType := vocab.DeviceTypeService
	td := thing.NewTD(directory.UpdateDirectoryCapability, title, deviceType)
	// TODO: add properties
	return td
}

func (svc *DirectoryService) handleTDEvent(event *hubclient.EventMessage) {
	args := directory.UpdateTDArgs{
		AgentID: event.AgentID,
		ThingID: event.ThingID,
		TDDoc:   event.Payload,
	}
	err := svc.updateDirSvc.UpdateTD(event.AgentID, args)
	if err != nil {
		slog.Error("handleTDEvent failed", "err", err)
	}
}

// Start the directory service and publish the service's own TD
// This subscribes to pubsub TD events and updates the directory.
func (svc *DirectoryService) Start() (err error) {

	// subscribe to TD events to add to the directory
	if svc.hc != nil {
		svc.tdSub, err = svc.hc.SubEvents(
			"", "", vocab.EventNameTD, svc.handleTDEvent)
	}

	// listen for requests
	bucket := svc.store.GetBucket(svc.tdBucketName)
	var capMap map[string]interface{}
	svc.readDirSvc, capMap = NewReadDirectoryService(svc.serviceID, bucket)
	svc.readSub, _ = hubclient.SubRPCCapability(svc.hc, directory.ReadDirectoryCapability, capMap)
	svc.updateDirSvc, capMap = NewUpdateDirectoryService(svc.serviceID, bucket)
	svc.updateSub, _ = hubclient.SubRPCCapability(svc.hc, directory.UpdateDirectoryCapability, capMap)

	// publish the TDs of this service
	if err == nil {
		myTD := svc.createReadDirTD()
		myTDJSON, _ := json.Marshal(myTD)
		err = svc.hc.PubEvent(
			directory.ReadDirectoryCapability, vocab.EventNameTD, myTDJSON)

		myTD = svc.createUpdateDirTD()
		myTDJSON, _ = json.Marshal(myTD)
		err = svc.hc.PubEvent(
			directory.UpdateDirectoryCapability, vocab.EventNameTD, myTDJSON)
	}

	// FIXME: register allowable roles with the auth service
	//  read: viewer and up
	//  update: manager, admin, service
	return err
}

// Stop the service
func (svc *DirectoryService) Stop() error {
	if svc.tdSub != nil {
		svc.tdSub.Unsubscribe()
		svc.tdSub = nil
	}
	if svc.readDirSvc != nil {
		svc.readDirSvc.Stop()
	}
	return nil
}

// NewDirectoryService creates an agent that provides capabilities to access TD documents
// The servicePubSub is optional and ignored when nil. It is used to subscribe to directory events and
// will be released on Stop.
//
//	store is an open bucket store for persisting the directory data.
//	hc is the hub client connection to use with this agent
func NewDirectoryService(
	store buckets.IBucketStore, hc hubclient.IHubClient) *DirectoryService {
	serviceID := directory.ServiceName
	//kvStore := kvbtree.NewKVStore(serviceID, thingStorePath)
	svc := &DirectoryService{
		hc:           hc,
		store:        store,
		serviceID:    serviceID,
		tdBucketName: TDBucketName,
	}
	return svc
}
