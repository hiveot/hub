package service

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/core/mqttmsgserver/jwtauth"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"strings"
	"sync"
)

// MqttAuthHook mochi-co MQTT broker authentication hook with
// validation methods.
type MqttAuthHook struct {
	mqtt.HookBase

	// map of known clients by ID for quick lookup during auth
	authClients map[string]msgserver.ClientAuthInfo

	// map of role to role permissions
	rolePermissions map[string][]msgserver.RolePermission

	authMux sync.RWMutex

	// server key for signing and verifying token signature
	signingKey    *ecdsa.PrivateKey
	signingKeyPub string

	// optionally require that the JWT token ID is that of a known user
	//jwtTokenMustBeKnownUser bool

	// ServicePermissions defines for each role the service capability that can be used
	servicePermissions map[string][]msgserver.RolePermission
}

// ApplyAuth apply update user authentication and authorization settings
func (hook *MqttAuthHook) ApplyAuth(clients []msgserver.ClientAuthInfo) error {
	authClients := map[string]msgserver.ClientAuthInfo{}
	for _, clientInfo := range clients {
		authClients[clientInfo.ClientID] = clientInfo
	}
	hook.authMux.Lock()
	hook.authClients = authClients
	hook.authMux.Unlock()

	return nil
}

// CreateKP creates a keypair for use in connecting or signing.
// This returns the key pair and its public key string.
func (hook *MqttAuthHook) CreateKP() (interface{}, string) {
	kp, _ := certs.CreateECDSAKeys()

	x509EncodedPub, _ := x509.MarshalPKIXPublicKey(&kp.PublicKey)

	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return kp, string(pemEncodedPub)
}

// CreateToken creates a new JWT authtoken for a client.
func (hook *MqttAuthHook) CreateToken(authInfo msgserver.ClientAuthInfo) (token string, err error) {
	token, err = jwtauth.CreateToken(authInfo, hook.signingKey)
	return token, err
}

// GetClientAuth returns the client auth info for the given ID
// This returns an error if the client is not found
func (hook *MqttAuthHook) GetClientAuth(clientID string) (msgserver.ClientAuthInfo, error) {
	hook.authMux.RLock()
	clientAuth, found := hook.authClients[clientID]
	hook.authMux.RUnlock()
	if !found {
		return clientAuth, fmt.Errorf("client '%s' not known", clientID)
	}
	return clientAuth, nil
}

// Init configures the hook with the auth config
func (hook *MqttAuthHook) Init(config any) error {
	return nil
}

// OnConnectAuthenticate returns true if the connecting client provides proof of its identity.
func (hook *MqttAuthHook) OnConnectAuthenticate(cl *mqtt.Client, pk packets.Packet) bool {
	clientID := string(cl.Properties.Username)
	// Multiple sessions by the same user will have different cid's.
	cid := pk.Connect.ClientIdentifier

	slog.Info("OnConnectAuthenticate",
		slog.String("clientID", clientID),
		slog.String("cid", cid),
		slog.Int("protocolVersion", int(cl.Properties.ProtocolVersion)))

	// Accept auth if a TLS connection with client cert is provided
	// The cert CN must be the clientID
	tlsConn, ok := cl.Net.Conn.(*tls.Conn)
	if ok {
		cState := tlsConn.ConnectionState()
		peerCerts := cState.PeerCertificates
		if len(peerCerts) > 0 {
			certID := peerCerts[0].Subject.CommonName
			if certID == clientID {
				return true
			}
		}
	}

	// verify authentication using password or token
	// step 1: credentials must be provided. Password contains password or token.
	if pk.Connect.PasswordFlag == false || len(pk.Connect.Password) == 0 {
		slog.Info("OnConnectAuthenticate: missing auth credentials",
			slog.String("cid", cid))
		return false
	}
	// step 2: determine what credential are provided, password or jwt
	// simply try if jwt token validation succeeds
	jwtString := string(pk.Connect.Password)
	authInfo, err := hook.ValidateToken(clientID, jwtString, "", "")
	if err == nil {
		// a valid JWT token
		_ = authInfo
		return true
	}
	slog.Info("OnConnectAuthenticate. Not a jwt token.", "err", err.Error())

	// step 3: password authentication, the user must be known
	authInfo, found := hook.authClients[clientID]
	if !found {
		slog.Info("OnConnectAuthenticate: unknown client",
			slog.String("clientID", clientID),
			slog.String("cid", cid))
		return false
	}
	if authInfo.PasswordHash != "" {
		// verify password
		err := bcrypt.CompareHashAndPassword([]byte(authInfo.PasswordHash), pk.Connect.Password)
		if err != nil {
			slog.Info("OnConnectAuthenticate: invalid password",
				"cid", cid,
				"net.remote", cl.Net.Remote)
			return false
		}
		slog.Info("OnConnectAuthenticate: password login success",
			"cid", cid,
			"net.remote", cl.Net.Remote)
		return true
	}
	// credentials provided but unable to match it
	slog.Info("OnConnectAuthenticate: invalid credentials",
		"cid", cid,
		"net.remote", cl.Net.Remote)
	//cl.Properties.Props.
	return false
}

// OnACLCheck returns true if the connecting client has matching read or write access to subscribe
// or publish to a given topic.
func (hook *MqttAuthHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {

	cid := cl.ID
	clientID := string(cl.Properties.Username)

	// todo: on connect, store role permissions in client session
	prof, err := hook.GetClientAuth(clientID)
	if err != nil {
		slog.Info("OnACLCheck: Unknown client",
			slog.String("clientID", clientID),
			slog.String("cid", cid))
		return false
	}

	// devices and services can publish a reply to any inbox
	//if write && strings.HasPrefix(topic, "_INBOX/") {
	//	if prof.ClientType == auth.ClientTypeDevice || prof.ClientType == auth.ClientTypeService {
	//		return true
	//	}
	//}
	// all clients can subscribe to their own inbox
	if !write && strings.HasPrefix(topic, "_INBOX/"+clientID) {
		return true
	}

	msgType, deviceID, thingID, name, senderID, err :=
		mqtthubclient.SplitTopic(topic)
	if err != nil {
		// invalid topic format.
		return false
	}
	_ = msgType
	_ = senderID

	//err := hook.hasRolePermissions(prof, topic)

	if hook.rolePermissions == nil {
		slog.Error("OnACLCheck: Role permissions not set")
		return false
	}

	rolePerm, found := hook.rolePermissions[prof.Role]
	if !found {
		slog.Info("OnACLCheck: Unknown role",
			slog.String("role", prof.Role),
			slog.String("clientID", clientID))
		return false
	}
	// publishing actions requires a valid client ID
	loginID := string(cl.Properties.Username)
	if loginID == "" {
		slog.Info("OnACLCheck: missing client ID for CID", "cid", cl.ID)
		return false
	}

	// include role permissions for individual services
	hook.authMux.RLock()
	sp, found := hook.servicePermissions[prof.Role]
	hook.authMux.RUnlock()
	if found {
		rolePerm = append(rolePerm, sp...)
	}
	for _, perm := range rolePerm {
		// when write, must allow pub, otherwise must allow sub
		if ((write && perm.AllowPub) || (!write && perm.AllowSub)) &&
			(perm.MsgType == "" || perm.MsgType == msgType) &&
			(perm.SourceID == "" || perm.SourceID == deviceID) &&
			(perm.ThingID == "" || perm.ThingID == thingID) &&
			(perm.MsgName == "" || perm.MsgName == name) {
			return true
		}
	}

	// customized perm
	if prof.Role == auth.ClientRoleDevice {
		if prof.ClientID != deviceID {
			slog.Info("Device can only pub/sub on its own things",
				slog.String("deviceID", clientID),
				slog.String("cid", cid),
				slog.String("topic", topic))
			return false
		}
	}

	slog.Info("OnAclCheck. success",
		slog.String("clientID", clientID),
		slog.String("CID", cl.ID),
		slog.String("topic", topic),
		slog.Bool("pub", write))
	return true
}

// Provides indicates which hook methods this hook provides.
func (hook *MqttAuthHook) Provides(b byte) bool {
	return bytes.Contains([]byte{
		mqtt.OnConnectAuthenticate,
		mqtt.OnACLCheck,
	}, []byte{b})
}

// SetRolePermissions applies the given permissions.
// rolePerms is a map of [role] to a list of permissions that role has.
// A default set of permissions for predefined roles is available in the auth api.
func (hook *MqttAuthHook) SetRolePermissions(
	rolePerms map[string][]msgserver.RolePermission) {

	hook.authMux.Lock()
	hook.rolePermissions = rolePerms
	hook.authMux.Unlock()
}

func (hook *MqttAuthHook) SetServicePermissions(
	serviceID string, capability string, roles []string) {
	slog.Info("SetServicePermissions",
		slog.String("serviceID", serviceID),
		slog.String("capability", capability))

	hook.authMux.Lock()
	for _, role := range roles {
		// add the role if needed
		rp := hook.servicePermissions[role]
		if rp == nil {
			rp = []msgserver.RolePermission{}
		}
		rp = append(rp, msgserver.RolePermission{
			Prefix:   "svc",
			SourceID: serviceID,
			ThingID:  capability,
			MsgType:  vocab.MessageTypeAction,
			MsgName:  "", // all methods of the capability can be used
			AllowPub: true,
			AllowSub: false,
		})
		hook.servicePermissions[role] = rp
	}
	hook.authMux.Unlock()
}

// ValidateToken verifies the given JWT token and returns its claims.
// optionally verify the signed nonce using the client's public key.
// This returns the auth info stored in the token.
func (hook *MqttAuthHook) ValidateToken(
	clientID string, token string, signedNonce string, nonce string) (
	authInfo msgserver.ClientAuthInfo, err error) {

	return jwtauth.ValidateToken(clientID, token, hook.signingKey, signedNonce, nonce)
}

func (hook *MqttAuthHook) ValidatePassword(
	loginID string, password string) (info msgserver.ClientAuthInfo, err error) {
	cinfo, found := hook.authClients[loginID]
	if !found {
		return cinfo, fmt.Errorf("ValidatePassword: Unknown user '%s", loginID)
	}
	if info.PasswordHash == "" {
		return cinfo, fmt.Errorf("ValidatePassword: Invalid password for user '%s'", loginID)
	}

	// verify password
	err = bcrypt.CompareHashAndPassword([]byte(cinfo.PasswordHash), []byte(password))
	return cinfo, err
}

func NewMqttAuthHook(signingKey *ecdsa.PrivateKey) *MqttAuthHook {
	signingKeyPub, _ := x509.MarshalPKIXPublicKey(&signingKey.PublicKey)
	signingKeyPubStr := base64.StdEncoding.EncodeToString(signingKeyPub)
	hook := &MqttAuthHook{
		HookBase:           mqtt.HookBase{},
		authClients:        nil,
		rolePermissions:    nil,
		authMux:            sync.RWMutex{},
		signingKey:         signingKey,
		signingKeyPub:      signingKeyPubStr,
		servicePermissions: make(map[string][]msgserver.RolePermission),
	}
	return hook
}
