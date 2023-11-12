package service

import (
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/core/auth/authclient"
	"github.com/hiveot/hub/core/state/stateapi"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"log/slog"
	"path"
)

// StateService handles storage of client data records
type StateService struct {
	// Hub connection
	hc *hubclient.HubClient
	// handler rpc subscription
	sub transports.ISubscription
	// backend storage
	storeDir string
	store    buckets.IBucketStore
}

func (svc *StateService) Delete(ctx hubclient.ServiceContext, args *stateapi.DeleteArgs) (err error) {
	bucket := svc.store.GetBucket(ctx.SenderID)
	err = bucket.Delete(args.Key)
	_ = bucket.Close()
	return err
}

func (svc *StateService) Get(ctx hubclient.ServiceContext, args *stateapi.GetArgs) (resp *stateapi.GetResp, err error) {
	bucket := svc.store.GetBucket(ctx.SenderID)
	value, err := bucket.Get(args.Key)
	// bucket returns an error if key is not found.
	found := err == nil
	resp = &stateapi.GetResp{
		Key:   args.Key,
		Found: found,
		Value: string(value)}

	err2 := bucket.Close()
	if err == nil {
		err = err2
	}
	return resp, err
}

func (svc *StateService) GetMultiple(
	ctx hubclient.ServiceContext, args *stateapi.GetMultipleArgs) (resp *stateapi.GetMultipleResp, err error) {

	bucket := svc.store.GetBucket(ctx.SenderID)
	kvbyte, err := bucket.GetMultiple(args.Keys)
	err = bucket.Close()
	// convert values back to string
	kvstring := make(map[string]string)
	for k, v := range kvbyte {
		kvstring[k] = string(v)
	}

	resp = &stateapi.GetMultipleResp{KV: kvstring}
	return resp, err
}

func (svc *StateService) Set(
	ctx hubclient.ServiceContext, args *stateapi.SetArgs) (err error) {
	slog.Info("Set", slog.String("key", args.Key))
	bucket := svc.store.GetBucket(ctx.SenderID)
	// bucket returns an error if key is invalid
	err = bucket.Set(args.Key, []byte(args.Value))
	if err != nil {
		slog.Warn("Set; Invalid key", slog.String("key", args.Key))
	}
	_ = bucket.Close()
	return err
}

func (svc *StateService) SetMultiple(
	ctx hubclient.ServiceContext, args *stateapi.SetMultipleArgs) (err error) {
	slog.Info("SetMultiple", slog.Int("count", len(args.KV)))
	// convert to string :(
	storage := make(map[string][]byte)
	for k, v := range args.KV {
		storage[k] = []byte(v)
	}

	bucket := svc.store.GetBucket(ctx.SenderID)
	err = bucket.SetMultiple(storage)
	_ = bucket.Close()
	return err
}

// Start the service
// This sets the permission for roles (any) that can use the state store and opens the store
func (svc *StateService) Start(hc *hubclient.HubClient) (err error) {
	slog.Warn("Starting the state service", "clientID", hc.ClientID())
	svc.hc = hc
	storePath := path.Join(svc.storeDir, "state.kvbtree")
	svc.store = kvbtree.NewKVStore(storePath)

	// Set the required permissions for using this service
	// any user roles can read and write their state
	serviceProfile := authclient.NewProfileClient(svc.hc)
	err = serviceProfile.SetServicePermissions(stateapi.StorageCap, []string{
		authapi.ClientRoleViewer,
		authapi.ClientRoleOperator,
		authapi.ClientRoleManager,
		authapi.ClientRoleAdmin,
		authapi.ClientRoleDevice,
		authapi.ClientRoleService})
	if err != nil {
		return err
	}
	err = svc.store.Open()

	if err == nil {
		// register the handler
		svc.sub, err = svc.hc.SubRPCCapability(stateapi.StorageCap,
			map[string]interface{}{
				stateapi.DeleteMethod:      svc.Delete,
				stateapi.GetMethod:         svc.Get,
				stateapi.GetMultipleMethod: svc.GetMultiple,
				stateapi.SetMethod:         svc.Set,
				stateapi.SetMultipleMethod: svc.SetMultiple,
			})
	}

	return err
}

// Stop the service
func (svc *StateService) Stop() {
	slog.Warn("Stopping the state service")
	if svc.sub != nil {
		svc.sub.Unsubscribe()
		svc.sub = nil
	}

	_ = svc.store.Close()
}

// NewStateService creates a new service instance using the kvstore
func NewStateService(storeDir string) *StateService {

	svc := &StateService{
		storeDir: storeDir,
	}

	return svc
}
