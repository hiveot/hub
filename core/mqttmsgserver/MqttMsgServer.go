package mqttmsgserver

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"github.com/eclipse/paho.golang/packets"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/listeners"
	"net"
	"os"
	"sync"
)

var MqttInMemUDS = "@/mqttinmemuds"

// MqttMsgServer runs a MQTT broker using the Mochi-co embedded mqtt server.
// this implements the IMsgServer interface
type MqttMsgServer struct {
	// authhook handles authentication and authorization for the server and mochi-co
	// this carries the mochi auth hook
	MqttAuthHook

	Config *MqttServerConfig

	//// map of known clients by ID for quick lookup during auth
	//authClients map[string]msgserver.ClientAuthInfo
	//
	//// map of role to role permissions
	//rolePermissions map[string][]msgserver.RolePermission

	// serverURL the server is listening on
	serverURL string

	ms      *mqtt.Server
	authMux sync.RWMutex
}

// ClientURL is the URL used to connect to this server. This is set on Start
func (srv *MqttMsgServer) ClientURL() string {
	return srv.serverURL
}

// ConnectInProc establishes a connection to the server for core services.
// This connects in-process using the service key.
// Intended for the core services to connect to the local server.
//
//	serviceID of the connecting service. The ID must be a known ID
//	token is the service authentication token
func (srv *MqttMsgServer) ConnectInProc(serviceID string) (hc hubclient.IHubClient, err error) {

	hubCl := mqtthubclient.NewMqttHubClient(
		"", serviceID, srv.Config.CoreServiceKP, nil)

	conn, err := net.Dial("unix", MqttInMemUDS)
	if err != nil {
		return nil, err
	}
	safeConn := packets.NewThreadSafeConn(conn)
	// use an on-the-fly created token for the connection
	token, err := srv.CreateToken(msgserver.ClientAuthInfo{
		ClientID:     serviceID,
		ClientType:   auth.ClientTypeService,
		PubKey:       srv.Config.CoreServicePub,
		PasswordHash: "",
		Role:         auth.ClientRoleAdmin,
	})
	if err != nil {
		return nil, err
	}
	err = hubCl.ConnectWithConn(token, safeConn)

	return hubCl, err
}

// Start the MQTT server using the configuration provided with NewMqttMsgServer().
// This returns the URL to connect to the server or an error if startup failed.
func (srv *MqttMsgServer) Start() (serverURL string, err error) {
	srv.serverURL = "not started"

	// Require TLS for tcp and wss listeners
	if srv.Config.CaCert == nil || srv.Config.ServerTLS == nil {
		return "", fmt.Errorf("missing server or CA certificate")
	}

	_ = os.Remove(MqttInMemUDS)

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
	srv.ms.Options.Capabilities.MinimumProtocolVersion = 5
	_ = srv.ms.AddHook(&srv.MqttAuthHook, nil)

	// server listens on TCP with TLS
	if srv.Config.Port != 0 {
		tcpLis := listeners.NewTCP("tcp1",
			fmt.Sprintf(":%d", srv.Config.Port),
			&listeners.Config{
				TLSConfig: tlsConfig,
			})

		err = srv.ms.AddListener(tcpLis)
		if err != nil {
			return "", err
		}
		//srv.serverURL = fmt.Sprintf("tls://localhost:%d", srv.Config.Port)
		srv.serverURL = fmt.Sprintf("tcp://localhost:%d", srv.Config.Port)
	}
	// server listens on Websocket with TLS
	if srv.Config.WSPort != 0 {
		wsLis := listeners.NewWebsocket("ws1",
			fmt.Sprintf(":%d", srv.Config.WSPort),
			&listeners.Config{TLSConfig: tlsConfig})
		err = srv.ms.AddListener(wsLis)
		if err != nil {
			return "", err
		}
	}
	// listen on UDS for local connections
	// todo: does @/path prefix creates an in-memory pipe
	inmemLis := listeners.NewUnixSock("inmem", MqttInMemUDS)
	err = srv.ms.AddListener(inmemLis)
	if err != nil {
		return "", err
	}

	err = srv.ms.Serve()
	if err != nil {
		return "", err
	}
	return srv.serverURL, nil
}

// Stop the server
func (srv *MqttMsgServer) Stop() {
	if srv.ms != nil {
		_ = srv.ms.Close()
		srv.ms = nil
	}
}

// NewMqttMsgServer creates a new instance of the Hub MQTT broker.
//
//	cfg contains the server configuration. Setup must have been called successfully first.
//	perms contain the map of roles and permissions. See SetRolePermissions for more detail.
func NewMqttMsgServer(cfg *MqttServerConfig, perms map[string][]msgserver.RolePermission) *MqttMsgServer {
	signingKeyPub, _ := x509.MarshalPKIXPublicKey(&cfg.ServerKey.PublicKey)
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)
	srv := &MqttMsgServer{
		MqttAuthHook: MqttAuthHook{
			HookBase:           mqtt.HookBase{},
			authClients:        nil,
			rolePermissions:    nil,
			authMux:            sync.RWMutex{},
			signingKey:         cfg.ServerKey,
			signingKeyPub:      signingKeyPubStr,
			servicePermissions: make(map[string][]msgserver.RolePermission),
		},
		Config: cfg,
	}
	srv.MqttAuthHook.SetRolePermissions(perms)
	return srv
}
