// Package main with the thing directory store
package main

import (
	"context"
	"github.com/hiveot/hub/lib/hubclient"
	"net"
	"path/filepath"

	"github.com/hiveot/hub/lib/listener"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/hiveot/hub/pkg/bucketstore/kvbtree"
	"github.com/hiveot/hub/pkg/directory"
	"github.com/hiveot/hub/pkg/directory/capnpserver"
	"github.com/hiveot/hub/pkg/directory/service"
	"github.com/hiveot/hub/pkg/pubsub/capnpclient"
)

// name of the storage file
const storeFile = "directorystore.json"

// Connect the service
func main() {
	var fullUrl = "" // TODO, from config

	ctx := context.Background()
	serviceID := directory.ServiceName
	f, clientCert, caCert := svcconfig.SetupFolderConfig(directory.ServiceName)

	// the service uses the bucket store to store directory entries
	storePath := filepath.Join(f.Stores, directory.ServiceName, storeFile)

	// Initialize the resolver client and marshallers to access the certificate and pubsub services
	// This allows them to live anywhere.
	//resolver.RegisterCapnpMarshaller[pubsub.IPubSubService](capnpclient.NewPubSubCapnpClient, "")

	// the resolver client is a proxy for all connected services including pubsub
	fullUrl = hubclient.LocateHub("", 0)
	capClient, err := hubclient.ConnectWithCapnpTCP(fullUrl, clientCert, caCert)
	pubSubClient := capnpclient.NewPubSubCapnpClient(capClient)
	if pubSubClient == nil {
		panic("can't connect to pubsub")
	}
	svcPubSub, err := pubSubClient.CapServicePubSub(ctx, serviceID)

	store := kvbtree.NewKVStore(directory.ServiceName, storePath)
	err = store.Open()
	if err != nil {
		panic("unable to open the directory store")
	}
	svc := service.NewDirectoryService(serviceID, store, svcPubSub)

	listener.RunService(directory.ServiceName, f.SocketPath,
		func(ctx context.Context, lis net.Listener) error {
			// startup
			err := svc.Start()
			if err == nil {
				err = capnpserver.StartDirectoryServiceCapnpServer(svc, lis)
			}
			return err
		}, func() error {
			// shutdown
			svc.Stop()
			pubSubClient.Release()
			err = store.Close()
			return err
		})
}
