package authzadapter

import (
	"fmt"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/authz/authzservice"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/hiveot/hub/core/msgserver/natsserver"
)

// StartNatsAuthz starts the authorization service for use with the NATS nkey server
func StartNatsAuthz(natsSrv *natsserver.NatsNKeyServer, aclFilePath string) (authz.IAuthz, error) {
	aclStore := authzservice.NewAuthzFileStore(aclFilePath)

	// use the in-proc message handler connection with the message bus to:
	nc1, err := natsSrv.ConnectInProc("authz", nil)
	hc1, _ := natshubclient.ConnectWithNC(nc1, "authz")
	if err != nil {
		return nil, fmt.Errorf("can't connect to server: " + err.Error())
	}
	// the adapter applies authz configuration to configure groups using jetstream
	authzAdpt := NewNatsAuthzAdapter(aclStore, natsSrv)
	if err != nil {
		return nil, fmt.Errorf("can't initialize nats authz binding: " + err.Error())
	}
	// the service subscribes to action requests using the service client connection,
	// uses the store to persist groups, and uses the adapter to apply changes to the server
	authzSvc := authzservice.NewAuthzService(aclStore, authzAdpt, hc1)
	err = authzSvc.Start()
	if err != nil {
		return nil, err
	}
	return authzSvc, nil
}
