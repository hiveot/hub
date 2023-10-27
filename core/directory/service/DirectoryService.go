package service

import (
	"encoding/json"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/core/directory/directoryapi"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/thing"
	"github.com/hiveot/hub/lib/vocab"
	"log/slog"
)

const TDBucketName = "td"

// DirectoryService is a wrapper around the internal store store
// This implements the IDirectory interface
type DirectoryService struct {
	hc           *hubclient.HubClient
	store        buckets.IBucketStore
	agentID      string // thingID of the service instance
	tdBucketName string
	tdBucket     buckets.IBucket

	// td event subscription
	tdSub transports.ISubscription

	// capabilities and subscriptions
	readDirSvc *ReadDirectoryService
	//readSub      hubclient.ISubscription
	updateDirSvc *UpdateDirectoryService
	//updateSub    hubclient.ISubscription
}

// handleTDEvent stores a received Thing TD document
func (svc *DirectoryService) handleTDEvent(event *thing.ThingValue) {
	args := directoryapi.UpdateTDArgs{
		AgentID: event.AgentID,
		ThingID: event.ThingID,
		TDDoc:   event.Data,
	}
	ctx := hubclient.ServiceContext{SenderID: event.AgentID}
	err := svc.updateDirSvc.UpdateTD(ctx, args)
	if err != nil {
		slog.Error("handleTDEvent failed", "err", err)
	}
}

// Start the directory service and publish the service's own TD
// This subscribes to pubsub TD events and updates the directory.
func (svc *DirectoryService) Start() (err error) {

	// listen for requests
	tdBucket := svc.store.GetBucket(svc.tdBucketName)
	svc.tdBucket = tdBucket

	svc.readDirSvc, err = StartReadDirectoryService(svc.hc, tdBucket)
	if err == nil {
		svc.updateDirSvc, err = StartUpdateDirectoryService(svc.hc, tdBucket)
	}
	// subscribe to TD events to add to the directory
	if svc.hc != nil {
		svc.tdSub, err = svc.hc.SubEvents(
			"", "", vocab.EventNameTD, svc.handleTDEvent)
	}
	myProfile := authclient.NewProfileClient(svc.hc)

	// Set the required permissions for using this service
	// any user roles can view the directory
	err = myProfile.SetServicePermissions(directoryapi.ReadDirectoryCap, []string{
		authapi.ClientRoleViewer,
		authapi.ClientRoleOperator,
		authapi.ClientRoleManager,
		authapi.ClientRoleAdmin,
		authapi.ClientRoleService})
	if err == nil {
		// only admin role can manage the directory
		err = myProfile.SetServicePermissions(directoryapi.UpdateDirectoryCap, []string{authapi.ClientRoleAdmin})
	}
	// last, publish a TD for each service capability and set allowable roles
	if err == nil {
		myTD := svc.updateDirSvc.CreateUpdateDirTD()
		myTDJSON, _ := json.Marshal(myTD)
		err = svc.hc.PubEvent(directoryapi.UpdateDirectoryCap, vocab.EventNameTD, myTDJSON)
	}
	if err == nil {
		// last, publish my TD
		myTD := svc.readDirSvc.CreateReadDirTD()
		myTDJSON, _ := json.Marshal(myTD)
		err = svc.hc.PubEvent(directoryapi.ReadDirectoryCap, vocab.EventNameTD, myTDJSON)
	}

	return err
}

// Stop the service
func (svc *DirectoryService) Stop() {
	if svc.tdSub != nil {
		svc.tdSub.Unsubscribe()
		svc.tdSub = nil
	}
	if svc.updateDirSvc != nil {
		svc.updateDirSvc.Stop()
	}
	if svc.readDirSvc != nil {
		svc.readDirSvc.Stop()
	}
	if svc.tdBucket != nil {
		_ = svc.tdBucket.Close()
	}
}

// NewDirectoryService creates an agent that provides capabilities to access TD documents
// The servicePubSub is optional and ignored when nil. It is used to subscribe to directory events and
// will be released on Stop.
//
//	hc is the hub client connection to use with this agent. Its ID is used as the agentID that provides the capability.
//	store is an open store store containing the directory data.
func NewDirectoryService(
	hc *hubclient.HubClient, store buckets.IBucketStore) *DirectoryService {
	//kvStore := kvbtree.NewKVStore(agentID, thingStorePath)
	svc := &DirectoryService{
		hc:           hc,
		store:        store,
		agentID:      hc.ClientID(),
		tdBucketName: TDBucketName,
	}
	return svc
}
