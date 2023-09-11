package mqttmsgserver

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"github.com/eclipse/paho.golang/packets"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/hubclient/mqtthubclient"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/hooks/auth"
	"github.com/mochi-mqtt/server/v2/listeners"
	"net"
	"os"
)

var inProcAddr = "@/mqttinproc"

//var inProcAddr = "/tmp/mqttinproc"

// MqttMsgServer runs a MQTT broker using the Mochi-co embedded mqtt server.
// this implements the IMsgServer interface
type MqttMsgServer struct {
	Config *MqttServerConfig

	// map of known clients by ID for quick lookup during auth
	authClients map[string]msgserver.ClientAuthInfo

	// map of role to role permissions
	rolePermissions map[string][]msgserver.RolePermission

	// clientURL the server is listening on
	clientURL string

	//
	ms *mqtt.Server
}

// ClientURL is the URL used to connect to this server. This is set on Start
func (srv *MqttMsgServer) ClientURL() string {
	return srv.clientURL
}

// ConnectInProcNC establishes a connection to the server for core services.
// This connects in-process using the service key.
// Intended for the core services to connect to the server.
//
//	serviceID of the connecting service
//	clientKey is optional alternate key or nil to use the built-in core service ID
func (srv *MqttMsgServer) ConnectInProcNC() (net.Conn, error) {

	//if clientKey == nil {
	//	clientKey = srv.Config.CoreServiceKP
	//}
	//// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.Config.CaCert != nil {
		caCertPool.AddCert(srv.Config.CaCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.Config.CaCert == nil,
	}
	//serviceKeyPub := &clientKey.PublicKey
	clientURL := "localhost:8441"
	nc, err := tls.Dial("tcp", clientURL, tlsConfig)
	//slog.Info("ConnectInProc", "serviceID", serviceID, "pubkey", serviceKeyPub)

	//nc, err := net.Dial("unix", inProcAddr)
	return nc, err
}

// ConnectInProc establishes a connection to the server for core services.
// This connects in-process using the service key.
// Intended for the core services to connect to the server.
//
//	serviceID of the connecting service
func (srv *MqttMsgServer) ConnectInProc(serviceID string) (hubclient.IHubClient, error) {

	conn, err := net.Dial("unix", inProcAddr)
	if err != nil {
		return nil, err
	}
	safeConn := packets.NewThreadSafeConn(conn)
	hc, err := mqtthubclient.ConnectToBroker(serviceID, "", safeConn)

	return hc, err
}

// Start the NATS server with the given configuration and create an event ingress stream
//
//	Config.Setup must have been called first.
func (srv *MqttMsgServer) Start() (clientURL string, err error) {
	srv.clientURL = "not implemented"

	// Require TLS for tcp and wss listeners
	if srv.Config.CaCert == nil || srv.Config.ServerTLS == nil {
		return "", fmt.Errorf("missing server or CA certificate")
	}

	_ = os.Remove(inProcAddr)

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(srv.Config.CaCert)
	clientCertList := []tls.Certificate{*srv.Config.ServerTLS}
	tlsConfig := &tls.Config{
		ServerName:   "HiveOT Hub",
		ClientCAs:    caCertPool,
		RootCAs:      caCertPool,
		Certificates: clientCertList,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS13,
	}

	srv.ms = mqtt.New(nil)
	_ = srv.ms.AddHook(new(auth.AllowHook), nil)
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
		//srv.clientURL = fmt.Sprintf("tls://localhost:%d", srv.Config.Port)
		srv.clientURL = fmt.Sprintf("tcp://localhost:%d", srv.Config.Port)
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
	inmemLis := listeners.NewUnixSock("inmem", inProcAddr)
	err = srv.ms.AddListener(inmemLis)
	if err != nil {
		return "", err
	}

	err = srv.ms.Serve()
	if err != nil {
		return "", err
	}
	return srv.clientURL, nil
}

// Stop the server
func (srv *MqttMsgServer) Stop() {
	if srv.ms != nil {
		_ = srv.ms.Close()
		srv.ms = nil
	}
}

// NewMqttMsgServer creates a new instance of the Hub MQTT broker.
func NewMqttMsgServer(cfg *MqttServerConfig, perms map[string][]msgserver.RolePermission) *MqttMsgServer {

	srv := &MqttMsgServer{Config: cfg, rolePermissions: perms}
	return srv
}
