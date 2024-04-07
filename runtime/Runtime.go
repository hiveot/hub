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

	authnStore  authn.IAuthnStore
	authnSvc    *service.AuthnService
	authzSvc    *authz.AuthzService
	dirSvc      *service2.DirectoryService
	msgRouter   *router.MessageRouter
	protocolMgr *protocols.ProtocolsManager
	valueSvc    *valueservice.ValueService
	valueStore  buckets.IBucketStore
}

func (r *Runtime) Start(env *plugin.AppEnvironment) error {
	err := r.cfg.Setup(env)

	if err != nil {
		return err
	}

	// startup
	var sessionAuth authn.IAuthenticator
	r.authnSvc, sessionAuth, r.authnStore, err = StartAuthnSvc(&r.cfg.Authn, env.CaCert)
	if err != nil {
		return err
	}

	r.authzSvc, err = StartAuthzSvc(&r.cfg.Authz, r.authnStore)
	if err != nil {
		return err
	}

	r.dirSvc, err = StartDirectorySvc(&r.cfg.Directory, env.StoresDir)
	if err != nil {
		return err
	}

	r.valueSvc, r.valueStore, err = StartValueSvc(&r.cfg.ValueStore, env.StoresDir)
	if err != nil {
		return err
	}

	// setup ingress routing; right now its quite simple
	r.msgRouter = router.NewMessageRouter(&r.cfg.Router)
	r.msgRouter.AddMessageTypeHandler(vocab.MessageTypeEvent, func(tv *things.ThingMessage) ([]byte, error) {
		_, _ = r.dirSvc.HandleEvent(tv)
		_, _ = r.valueSvc.HandleEvent(tv)
		return nil, nil
	})
	r.msgRouter.AddMessageTypeHandler(vocab.MessageTypeAction, func(tv *things.ThingMessage) ([]byte, error) {
		return nil, fmt.Errorf("not yet implemented")
	})
	r.msgRouter.AddMessageTypeHandler(vocab.MessageTypeRPC, func(tv *things.ThingMessage) ([]byte, error) {
		return nil, fmt.Errorf("not yet implemented")
	})

	r.protocolMgr, err = StartProtocolsManager(
		&r.cfg.Protocols, r.cfg.ServerKey, r.cfg.ServerCert, r.cfg.CaCert, sessionAuth, r.msgRouter.HandleMessage)
	return err
}

func StartAuthnSvc(cfg *authn.AuthnConfig, caCert *x509.Certificate) (
	svc *service.AuthnService, sessionAuth authn.IAuthenticator,
	store authn.IAuthnStore, err error) {

	// setup the Authentication service
	authStore := authnstore.NewAuthnFileStore(cfg.PasswordFile, cfg.Encryption)
	svc = service.NewAuthnService(cfg, authStore, caCert)
	sessionAuth, err = svc.Start()
	return svc, sessionAuth, authStore, err
}

func StartAuthzSvc(cfg *authz.AuthzConfig,
	authnStore authn.IAuthnStore) (svc *authz.AuthzService, err error) {

	// setup the Authentication service
	svc = authz.NewAuthzService(cfg, authnStore)
	err = svc.Start()
	return svc, err
}

func StartDirectorySvc(cfg *directory.DirectoryConfig, storesDir string) (
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
	}
	return svc, err
}

func StartProtocolsManager(cfg *protocols.ProtocolsConfig,
	serverKey keys.IHiveKey, serverCert *tls.Certificate, caCert *x509.Certificate,
	sessionAuth authn.IAuthenticator,
	handler func(tv *things.ThingMessage) ([]byte, error)) (
	svc *protocols.ProtocolsManager, err error) {

	pm := protocols.NewProtocolManager(
		cfg, serverKey, serverCert, caCert, sessionAuth, handler)

	if err == nil {
		err = pm.Start()
	}
	return pm, err
}

func StartValueSvc(cfg *valueservice.ValueStoreConfig, storesDir string) (
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
	}

	return valueSvc, valueStore, err
}

func (r *Runtime) Stop() {
	if r.protocolMgr != nil {
		r.protocolMgr.Stop()
	}
	if r.msgRouter != nil {
	}
	if r.valueSvc != nil {
		r.valueSvc.Stop()
	}
	if r.valueStore != nil {
		_ = r.valueStore.Close()
	}
	if r.dirSvc != nil {
		r.dirSvc.Stop()
	}
	if r.authzSvc != nil {
		r.authzSvc.Stop()
	}
	if r.authnSvc != nil {
		r.authnSvc.Stop()
	}
	if r.authnStore != nil {
		r.authnStore.Close()
	}
}

func NewRuntime(cfg *RuntimeConfig) *Runtime {
	r := &Runtime{cfg: cfg}
	return r
}
