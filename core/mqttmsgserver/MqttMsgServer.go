package mqttmsgserver

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/api/go/hubclient"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/hubclient/mqtthubclient"
	"golang.org/x/exp/slog"
)

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
func (srv *MqttMsgServer) ConnectInProcNC(serviceID string, clientKey *ecdsa.PrivateKey) (*tls.Conn, error) {

	if clientKey == nil {
		clientKey = srv.Config.CoreServiceKP
	}
	// If the server uses TLS then the in-process pipe connection is also upgrade to TLS.
	caCertPool := x509.NewCertPool()
	if srv.Config.CaCert != nil {
		caCertPool.AddCert(srv.Config.CaCert)
	}
	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: srv.Config.CaCert == nil,
	}
	serviceKeyPub := &clientKey.PublicKey
	nc, err := tls.Dial("tcp", srv.clientURL, tlsConfig)
	slog.Info("ConnectInProc", "serviceID", serviceID, "pubkey", serviceKeyPub)
	return nc, err
}

// ConnectInProc establishes a connection to the server for core services.
// This connects in-process using the service key.
// Intended for the core services to connect to the server.
//
//	serviceID of the connecting service
func (srv *MqttMsgServer) ConnectInProc(serviceID string) (hubclient.IHubClient, error) {

	nc, err := srv.ConnectInProcNC(serviceID, nil)
	if err != nil {
		return nil, err
	}
	hc, err := mqtthubclient.ConnectWithNC(nc)
	return hc, err
}

// Start the NATS server with the given configuration and create an event ingress stream
//
//	Config.Setup must have been called first.
func (srv *MqttMsgServer) Start() (clientURL string, err error) {
	srv.clientURL = "not implemented"
	panic("not implemented")
}

// Stop the server
func (srv *MqttMsgServer) Stop() {
	panic("not implemented")
}

// NewMqttMsgServer creates a new instance of the Hub MQTT broker.
func NewMqttMsgServer(cfg *MqttServerConfig, perms map[string][]msgserver.RolePermission) *MqttMsgServer {

	srv := &MqttMsgServer{Config: cfg, rolePermissions: perms}
	return srv
}
