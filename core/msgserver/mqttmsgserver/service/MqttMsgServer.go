package service

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/core/msgserver/mqttmsgserver"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"log/slog"
	"sync"
)

// MqttMsgServer runs a MQTT broker using the Mochi-co embedded mqtt server.
// this implements the IMsgServer interface
type MqttMsgServer struct {
	// authhook handles authentication and authorization for the server and mochi-co
	// this carries the mochi auth hook
	MqttAuthHook

	Config *mqttmsgserver.MqttServerConfig

	// urls the server is listening on
	tlsURL string
	wssURL string
	udsURL string

	ms      *mqtt.Server
	authMux sync.RWMutex
}

// GetServerURLs is the URL used to connect to this server. This is set on Start
func (srv *MqttMsgServer) GetServerURLs() (tsURL string, wssURL string, udsURL string) {
	return srv.tlsURL, srv.wssURL, srv.udsURL
}

func (srv *MqttMsgServer) Core() string {
	return "mqtt"
}

// Start the MQTT server using the configuration provided with NewMqttMsgServer().
// This returns the URL to connect to the server or an error if startup failed.
func (srv *MqttMsgServer) Start() error {
	var err error
	slog.Warn("Starting MqttMsgServer")
	// Require TLS for tcp and wss listeners
	if srv.Config.CaCert == nil || srv.Config.ServerTLS == nil {
		return fmt.Errorf("missing server or CA certificate")
	}
	hostAddr := srv.Config.Host
	srv.tlsURL = fmt.Sprintf("tcp://%s:%d", hostAddr, srv.Config.Port)
	srv.wssURL = fmt.Sprintf("wss://%s:%d", hostAddr, srv.Config.WSPort)
	srv.udsURL = srv.Config.InMemUDSName

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(srv.Config.CaCert)
	clientCertList := []tls.Certificate{*srv.Config.ServerTLS}
	tlsConfig := &tls.Config{
		ServerName:   "HiveOT Hub",
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		Certificates: clientCertList,
		ClientAuth:   tls.VerifyClientCertIfGiven, // allow client cert auth
		MinVersion:   tls.VersionTLS13,
	}

	srv.ms = mqtt.New(nil)
	//srv.ms.Options.Capabilities.MinimumProtocolVersion = 5
	srv.ms.Options.Capabilities.MinimumProtocolVersion = 3
	_ = srv.ms.AddHook(&srv.MqttAuthHook, nil)

	// TLS over TCP listener
	tlsAddr := fmt.Sprintf(":%d", srv.Config.Port)
	tcpLis := listeners.NewTCP("tcp1", tlsAddr,
		&listeners.Config{
			TLSConfig: tlsConfig,
		})
	err = srv.ms.AddListener(tcpLis)
	if err != nil {
		return err
	}

	// TLS over Websocket listener
	wssAddr := fmt.Sprintf(":%d", srv.Config.WSPort)
	wsLis := listeners.NewWebsocket("ws1", wssAddr,
		&listeners.Config{
			TLSConfig: tlsConfig,
		})
	err = srv.ms.AddListener(wsLis)
	if err != nil {
		return err
	}

	// listen on UDS for local connections.
	// A path starting with '@/' is an in-memory.
	inmemLis := listeners.NewUnixSock("inmem", srv.Config.InMemUDSName)
	err = srv.ms.AddListener(inmemLis)
	if err != nil {
		return err
	}

	err = srv.ms.Serve()
	if err != nil {
		return err
	}

	return nil
}

// Stop the server
func (srv *MqttMsgServer) Stop() {
	slog.Warn("Stopping MqttMsgServer")
	if srv.ms != nil {
		_ = srv.ms.Close()
		srv.ms = nil
	}
}

// NewMqttMsgServer creates a new instance of the Hub MQTT broker.
//
//	cfg contains the server configuration. Setup must have been called successfully first.
//	perms contain the map of roles and permissions. See SetRolePermissions for more detail.
func NewMqttMsgServer(cfg *mqttmsgserver.MqttServerConfig, perms map[string][]msgserver.RolePermission) *MqttMsgServer {
	srv := &MqttMsgServer{
		MqttAuthHook: *NewMqttAuthHook(cfg.ServerKey),
		Config:       cfg,
	}
	srv.MqttAuthHook.SetRolePermissions(perms)
	return srv
}
