package runtime

import (
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/buckets"
	"github.com/hiveot/hub/lib/buckets/kvbtree"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/lib/plugin"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/authn"
	"github.com/hiveot/hub/runtime/authn/authnhandler"
	"github.com/hiveot/hub/runtime/authn/service"
	"github.com/hiveot/hub/runtime/authz"
	"github.com/hiveot/hub/runtime/authz/authzhandler"
	"github.com/hiveot/hub/runtime/digitwin/digitwinhandler"
	service3 "github.com/hiveot/hub/runtime/digitwin/service"
	"github.com/hiveot/hub/runtime/protocols"
	"github.com/hiveot/hub/runtime/router"
	"path"
)

const DefaultDigiTwinStoreFilename = "digitwin.kvbtree"

// Runtime is the Hub runtime
type Runtime struct {
	cfg *RuntimeConfig

	//AuthnStore api.IAuthnStore
	AuthnSvc *service.AuthnService
	AuthzSvc *authz.AuthzService
	//DirSvc      *service2.DirectoryService
	DigiTwinSvc *service3.DigiTwinService
	MsgRouter   *router.MessageRouter
	ProtocolMgr *protocols.ProtocolsManager
	//ValueSvc    *valueservice.ValueService
	//ValueStore  buckets.IBucketStore
}

func (r *Runtime) Start(env *plugin.AppEnvironment) error {
	err := r.cfg.Setup(env)

	if err != nil {
		return err
	}

	// setup ingress routing; right now it is quite simple with a single event handler
	r.MsgRouter = router.NewMessageRouter(&r.cfg.Router)

	// startup
	r.AuthnSvc, err = StartAuthnSvc(&r.cfg.Authn, env.CaCert, r.MsgRouter)
	if err != nil {
		return err
	}
	//r.AuthnSvc, sessionAuth, r.AuthnStore,

	r.AuthzSvc, err = StartAuthzSvc(&r.cfg.Authz, r.AuthnSvc.AuthnStore, r.MsgRouter)
	if err != nil {
		return err
	}

	r.DigiTwinSvc, err = StartDigiTwinSvc(env.StoresDir, r.MsgRouter)
	if err != nil {
		return err
	}

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
		&r.cfg.Protocols, r.cfg.ServerKey, r.cfg.ServerCert, r.cfg.CaCert,
		r.AuthnSvc.SessionAuth, r.MsgRouter.HandleMessage)
	return err
}

func StartAuthnSvc(
	cfg *authn.AuthnConfig, caCert *x509.Certificate, r *router.MessageRouter) (
	svc *service.AuthnService, err error) {

	// setup the Authentication service
	svc, err = service.StartAuthnService(cfg)
	if err != nil {
		return nil, err
	}
	// add the messaging interface handler
	adminMsgHandler := authnhandler.NewAuthnAdminHandler(svc.AdminSvc)
	clientMsgHandler := authnhandler.NewAuthnUserHandler(svc.UserSvc)
	r.AddServiceHandler(api.AuthnAdminThingID, adminMsgHandler)
	r.AddServiceHandler(api.AuthnUserThingID, clientMsgHandler)
	return svc, err
}

func StartAuthzSvc(cfg *authz.AuthzConfig,
	authnStore api.IAuthnStore, r *router.MessageRouter) (svc *authz.AuthzService, err error) {

	// setup the Authentication service
	svc = authz.NewAuthzService(cfg, authnStore)
	err = svc.Start()

	// add the messaging interface handler
	msgHandler := authzhandler.NewAuthzHandler(svc)
	r.AddServiceHandler(api.AuthzThingID, msgHandler)
	return svc, err
}

func StartDigiTwinSvc(storesDir string, r *router.MessageRouter) (*service3.DigiTwinService, error) {

	var store buckets.IBucketStore
	storePath := path.Join(storesDir, "digitwin", DefaultDigiTwinStoreFilename)
	store = kvbtree.NewKVStore(storePath)
	err := store.Open()
	if err != nil {
		return nil, err
	}

	svc := service3.NewDigiTwinService(store)
	err = svc.Start()

	// add the messaging interface handler
	msgHandler := digitwinhandler.NewDigiTwinHandler(svc)
	r.AddServiceHandler(api.DigiTwinThingID, msgHandler)
	r.AddEventHandler(msgHandler)
	return svc, err
}

func StartProtocolsManager(cfg *protocols.ProtocolsConfig,
	serverKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	sessionAuth api.IAuthenticator,
	handler func(msg *things.ThingMessage) ([]byte, error)) (
	svc *protocols.ProtocolsManager, err error) {

	pm := protocols.NewProtocolManager(
		cfg, serverKey, serverCert, caCert, sessionAuth)

	if err == nil {
		err = pm.Start(handler)
	}
	return pm, err
}

//func StartValueSvc(cfg *valueservice.ValueStoreConfig, storesDir string, r *router.MessageRouter) (
//	valueSvc *valueservice.ValueService, valueStore buckets.IBucketStore, err error) {
//
//	if err == nil {
//		storeDir := path.Join(storesDir, "values")
//		valueStore, err = bucketstore.NewBucketStore(
//			storeDir, cfg.StoreFilename, buckets.BackendPebble)
//		if err != nil {
//			err = fmt.Errorf("can't open history bucket store: %w", err)
//		} else {
//			err = valueStore.Open()
//		}
//	}
//	if err == nil {
//		valueSvc = valueservice.NewThingValueService(cfg, valueStore)
//		err = valueSvc.Start()
//
//		// should these be registered through their TD? not right now
//		r.AddMessageHandler(vocab.MessageTypeEvent, api.ValueServiceID, vocab.EventTypeTD, valueSvc.HandleMessage)
//		r.AddMessageHandler(vocab.MessageTypeAction, api.ValueServiceID, "", valueSvc.HandleMessage)
//	}
//
//	return valueSvc, valueStore, err
//}

func (r *Runtime) Stop() {
	if r.ProtocolMgr != nil {
		r.ProtocolMgr.Stop()
	}
	if r.MsgRouter != nil {
	}
	//if r.ValueSvc != nil {
	//	r.ValueSvc.Stop()
	//}
	//if r.ValueStore != nil {
	//	_ = r.ValueStore.Close()
	//}
	if r.DigiTwinSvc != nil {
		r.DigiTwinSvc.Stop()
	}
	if r.AuthzSvc != nil {
		r.AuthzSvc.Stop()
	}
	if r.AuthnSvc != nil {
		r.AuthnSvc.Stop()
	}
}

func NewRuntime(cfg *RuntimeConfig) *Runtime {
	r := &Runtime{cfg: cfg}
	return r
}
