package authnservice

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/core/authn/authnstore"
	"github.com/hiveot/hub/core/hubclient"
)

// AuthnService creates a service handling both manage and user requests.
type AuthnService struct {
	mngBinding *AuthnManageBinding
	mngService *AuthnManageService
	usrBinding *AuthnUserBinding
	usrService *AuthnUserService
}

// Start the service and activate the binding to handle requests
func (svc *AuthnService) Start() error {
	err := svc.mngBinding.Start()
	if err == nil {
		err = svc.usrBinding.Start()
		if err != nil {
			svc.mngBinding.Stop()
		}
	}
	return err
}

// Stop the service and unsubscribe
func (svc *AuthnService) Stop() {
	svc.mngBinding.Stop()
	svc.usrBinding.Stop()
}

// NewAuthnService creates an authentication service instance
//
//	store is the client store to store authentication clients
//	tokenizer is the method of creating and validating JWT tokens
//	hc is the message bus connection used to subscribe to using bindings
func NewAuthnService(store authnstore.IAuthnStore, tokenizer authn.IAuthnTokenizer, hc hubclient.IHubClient) *AuthnService {
	mngSvc := NewAuthnManageService(store, tokenizer)
	mngBinding := NewAuthnManageBinding(mngSvc, hc)
	userSvc := NewAuthnUserService(store, tokenizer, nil)
	userBinding := NewAuthnUserBinding(userSvc, hc)

	authnSvc := &AuthnService{
		mngService: mngSvc,
		mngBinding: mngBinding,
		usrService: userSvc,
		usrBinding: userBinding,
	}
	return authnSvc
}
