// Package main with the history store
package main

import (
	"context"
	"github.com/sirupsen/logrus"
	"net"

	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/listener"
	"github.com/hiveot/hub/lib/svcconfig"
	"github.com/hiveot/hub/pkg/bucketstore/cmd"
	"github.com/hiveot/hub/pkg/history"
	"github.com/hiveot/hub/pkg/history/capnpserver"
	"github.com/hiveot/hub/pkg/history/config"
	"github.com/hiveot/hub/pkg/history/service"
	"github.com/hiveot/hub/pkg/pubsub/capnpclient"
)

// Connect the history store service
func main() {
	var fullUrl = "" // TODO, from config
	ctx := context.Background()
	f, clientCert, caCert := svcconfig.SetupFolderConfig(history.ServiceName)
	cfg := config.NewHistoryConfig(f.Stores)
	_ = f.LoadConfig(&cfg)

	// the service receives the events to store from pubsub. To obtain the pubsub capability
	// connect to the resolver or gateway service.
	fullUrl = hubclient.LocateHub("", 0)
	// the resolver client is a proxy for all connected services including pubsub
	capClient, err := hubclient.ConnectWithCapnpTCP(fullUrl, clientCert, caCert)
	pubSubClient := capnpclient.NewPubSubCapnpClient(capClient)
	svcPubSub, err := pubSubClient.CapServicePubSub(ctx, cfg.ServiceID)
	if err != nil {
		panic("can't connect to pubsub")
	}

	// the service uses the bucket store to store history
	store := cmd.NewBucketStore(cfg.Directory, cfg.ServiceID, cfg.Backend)
	err = store.Open()
	if err != nil {
		logrus.Panic("can't open history bucket store")
	}
	svc := service.NewHistoryService(&cfg, store, svcPubSub)

	listener.RunService(history.ServiceName, f.SocketPath,
		func(ctx context.Context, lis net.Listener) error {
			// startup
			err = svc.Start()
			if err == nil {
				err = capnpserver.StartHistoryServiceCapnpServer(svc, lis)
			}
			return err
		}, func() error {
			// shutdown
			err := svc.Stop()
			pubSubClient.Release()
			_ = store.Close()
			return err
		})

}
