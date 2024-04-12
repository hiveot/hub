package runtime

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/bucketstore"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authnstore"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/directory"
	service2 "github.com/hiveot/hub/runtime/directory/service"
	"github.com/hiveot/hub/runtime/protocols"
	"github.com/hiveot/hub/runtime/router"
	"github.com/hiveot/hub/runtime/valueservice"
	"path"
)

// Runtime is the Hub runtime
type Runtime struct {
	cfg *RuntimeConfig

	AuthnStore  api.IAuthnStore
	AuthnSvc    *service.AuthnService
	AuthzSvc    *authz.AuthzService
	DirSvc      *service2.DirectoryService
	MsgRouter   *router.MessageRouter
	ProtocolMgr *protocols.ProtocolsManager
	ValueSvc    *valueservice.ValueService
	ValueStore  buckets.IBucketStore
}

func (r *Runtime) Start(env *plugin.AppEnvironment) error {
	err := r.cfg.Setup(env)

	if err != nil {
		return err
	}

	// setup ingress routing; right now it is quite simple with a single event handler
	r.MsgRouter = router.NewMessageRouter(&r.cfg.Router)

	// startup
	var sessionAuth api.IAuthenticator
	r.AuthnSvc, sessionAuth, r.AuthnStore, err = StartAuthnSvc(
		&r.cfg.Authn, env.CaCert, r.MsgRouter)
	if err != nil {
		return err
	}

	r.AuthzSvc, err = StartAuthzSvc(&r.cfg.Authz, r.AuthnStore, r.MsgRouter)
	if err != nil {
		return err
	}

	r.DirSvc, err = StartDirectorySvc(&r.cfg.Directory, env.StoresDir, r.MsgRouter)
	if err != nil {
		return err
	}

	r.ValueSvc, r.ValueStore, err = StartValueSvc(&r.cfg.ValueStore, env.StoresDir, r.MsgRouter)
	if err != nil {
		return err
	}
	//
	//// setup ingress routing; right now it is quite simple with a single event handler
	//r.MsgRouter = router.NewMessageRouter(&r.cfg.Router)
	// TODO: add built-in services as separate event types, eg TD and Properties events
	//r.MsgRouter.AddEventHandler("", "", func(tv *things.ThingMessage) ([]byte, error) {
	//	_, _ = r.DirSvc.HandleEvent(tv)
	//	_, _ = r.ValueSvc.HandleEvent(tv)
	//	return nil, nil
	//})
	// pass to the default action handler
	// TODO: add built-in services as separate actions
	//r.MsgRouter.AddActionHandler("", "", func(tv *things.ThingMessage) ([]byte, error) {
	//	return nil, fmt.Errorf("not yet implemented")
	//})

	r.ProtocolMgr, err = StartProtocolsManager(
		&r.cfg.Protocols, r.cfg.ServerKey, r.cfg.ServerCert, r.cfg.CaCert, sessionAuth,
		r.MsgRouter.HandleMessage)
	return err
}

func StartAuthnSvc(cfg *authn.AuthnConfig, caCert *x509.Certificate, r *router.MessageRouter) (
	svc *service.AuthnService, sessionAuth api.IAuthenticator,
	store api.IAuthnStore, err error) {

	// setup the Authentication service
	authStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	svc = service.NewAuthnService(cfg, authStore, caCert)
	sessionAuth, err = svc.Start()
	//r.AddEventHandler(api.AuthnServiceID, vocab.EventTypeTD, svc.HandleEvent)
	r.AddActionHandler(api.AuthnServiceID, "", svc.HandleActions)
	return svc, sessionAuth, authStore, err
}

func StartAuthzSvc(cfg *authz.AuthzConfig,
	authnStore api.IAuthnStore, r *router.MessageRouter) (svc *authz.AuthzService, err error) {

	// setup the Authentication service
	svc = authz.NewAuthzService(cfg, authnStore)
	err = svc.Start()
	//r.AddEventHandler(api.AuthzServiceID, vocab.EventTypeTD, svc.HandleEvent)
	//r.AddActionHandler(api.AuthzServiceID, "", svc.HandleRPCRequests)
	return svc, err
}

func StartDirectorySvc(cfg *directory.DirectoryConfig, storesDir string, r *router.MessageRouter) (
	svc *service2.DirectoryService, err error) {

	var dirStore buckets.IBucketStore
	if err == nil {
		dirStorePath := path.Join(storesDir, "directory", cfg.StoreFilename)
		dirStore = kvbtree.NewKVStore(dirStorePath)
		err = dirStore.Open()
	}
	if err == nil {
		svc = service2.NewDirectoryService(cfg, dirStore)
		err = svc.Start()
		// should these be registered through their TD? not right now
		r.AddEventHandler(api.DirectoryServiceID, vocab.EventTypeTD, svc.HandleTDEvent)
		r.AddActionHandler(api.DirectoryServiceID, "", svc.HandleActions)
	}
	return svc, err
}

func StartProtocolsManager(cfg *protocols.ProtocolsConfig,
	serverKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	sessionAuth api.IAuthenticator,
	handler func(msg *things.ThingMessage) ([]byte, error)) (
	svc *protocols.ProtocolsManager, err error) {

	pm := protocols.NewProtocolManager(
		cfg, serverKey, serverCert, caCert, sessionAuth, handler)

	if err == nil {
		err = pm.Start()
	}
	return pm, err
}

func StartValueSvc(cfg *valueservice.ValueStoreConfig, storesDir string, r *router.MessageRouter) (
	valueSvc *valueservice.ValueService, valueStore buckets.IBucketStore, err error) {

	if err == nil {
		storeDir := path.Join(storesDir, "values")
		valueStore, err = bucketstore.NewBucketStore(
			storeDir, cfg.StoreFilename, buckets.BackendPebble)
		if err != nil {
			err = fmt.Errorf("can't open history bucket store: %w", err)
		} else {
			err = valueStore.Open()
		}
	}
	if err == nil {
		valueSvc = valueservice.NewThingValueService(cfg, valueStore)
		err = valueSvc.Start()

		// should these be registered through their TD? not right now
		r.AddEventHandler(api.ValueServiceID, vocab.EventTypeTD, valueSvc.HandleEvent)
		r.AddActionHandler(api.ValueServiceID, "", valueSvc.HandleAction)
	}

	return valueSvc, valueStore, err
}

func (r *Runtime) Stop() {
	if r.ProtocolMgr != nil {
		r.ProtocolMgr.Stop()
	}
	if r.MsgRouter != nil {
	}
	if r.ValueSvc != nil {
		r.ValueSvc.Stop()
	}
	if r.ValueStore != nil {
		_ = r.ValueStore.Close()
	}
	if r.DirSvc != nil {
		r.DirSvc.Stop()
	}
	if r.AuthzSvc != nil {
		r.AuthzSvc.Stop()
	}
	if r.AuthnSvc != nil {
		r.AuthnSvc.Stop()
	}
	if r.AuthnStore != nil {
		r.AuthnStore.Close()
	}
}

func NewRuntime(cfg *RuntimeConfig) *Runtime {
	r := &Runtime{cfg: cfg}
	return r
}
