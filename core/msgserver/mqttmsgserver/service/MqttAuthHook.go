package service

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient/transports"
	"github.com/hiveot/hub/lib/hubclient/transports/mqtttransport"
	"github.com/hiveot/hub/lib/keys"
	"github.com/hiveot/hub/runtime/authn/jwtauth"
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
	signingKey keys.IHiveKey

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

// CreateKeyPair creates a keypair for use in connecting or signing.
// NOTE: intended for testing. Might be deprecated in the future.
func (hook *MqttAuthHook) CreateKeyPair() (kp keys.IHiveKey) {
	k := keys.NewKey(keys.KeyTypeECDSA)
	return k
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

// GetRolePermissions returns the role permissions for the given clientID
func (hook *MqttAuthHook) GetRolePermissions(role string, clientID string) ([]msgserver.RolePermission, bool) {
	// TODO: speed things up a bit by pre-calculating on login
	// take the role's permissions
	rolePerm, found := hook.rolePermissions[role]

	if !found {
		return nil, false
	}
	// add service permissions for this role
	hook.authMux.RLock()
	sp, found := hook.servicePermissions[role]
	hook.authMux.RUnlock()
	if found {
		rolePerm = append(rolePerm, sp...)
	}

	return rolePerm, true
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
		slog.Int("protocolVersion", int(cl.Properties.ProtocolVersion)),
		slog.String("net.remote", cl.Net.Remote),
	)

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
		slog.Warn("OnConnectAuthenticate: missing auth credentials",
			slog.String("cid", cid))
		return false
	}
	// step 2: determine what credential are provided, password or jwt
	// simply try if jwt token validation succeeds
	jwtString := string(pk.Connect.Password)
	err := hook.ValidateToken(clientID, jwtString, "", "")
	if err == nil {
		// a valid JWT token
		slog.Info("Login success using a valid jwt token.",
			slog.String("clientID", string(pk.Connect.Username)))
		return true
	}
	slog.Debug("OnConnectAuthenticate. Not a jwt token.", slog.String("err", err.Error()))

	// step 3: password authentication, the user must be known
	authInfo, found := hook.authClients[clientID]
	if !found {
		slog.Warn("OnConnectAuthenticate: unknown client",
			slog.String("clientID", clientID),
			slog.String("cid", cid))
		return false
	}
	if authInfo.PasswordHash != "" {
		// verify password
		err := bcrypt.CompareHashAndPassword([]byte(authInfo.PasswordHash), pk.Connect.Password)
		if err != nil {
			slog.Warn("OnConnectAuthenticate: invalid password",
				"cid", cid,
				"net.remote", cl.Net.Remote)
			return false
		}
		slog.Info("Login success using a valid password", "cid", cid)
		return true
	}
	// credentials provided but unable to match it
	slog.Warn("OnConnectAuthenticate: invalid credentials",
		"cid", cid,
		"net.remote", cl.Net.Remote)
	//cl.Properties.Props.
	return false
}

// OnACLCheck returns true if the connecting client has matching read or write access to subscribe
// or publish to a given topic.
// Embedded rules are:
//
//	allow sub to user's own _INBOX
//	allow pub to any _INBOX
//	senderID must match loginID in all other messages
func (hook *MqttAuthHook) OnACLCheck(cl *mqtt.Client, topic string, write bool) bool {

	cid := cl.ID
	loginID := string(cl.Properties.Username)
	if loginID == "" {
		slog.Info("OnACLCheck: missing client ID for CID", "cid", cl.ID)
		return false
	}

	// todo: on connect, store role permissions in client session
	prof, err := hook.GetClientAuth(loginID)
	if err != nil {
		slog.Info("OnACLCheck: Unknown client",
			slog.String("clientID", loginID),
			slog.String("cid", cid))
		return false
	}

	// 1. INBOX rules are embedded

	// all clients can subscribe to their own inbox
	if !write && strings.HasPrefix(topic, transports.MessageTypeINBOX+"/"+loginID) {
		return true
	}
	// anyone can write to another inbox
	if write && strings.HasPrefix(topic, transports.MessageTypeINBOX) {
		return true
	}

	// 2. Break down the topic to match it with the permissions
	msgType, agentID, thingID, name, senderID, err :=
		mqtttransport.SplitTopic(topic)
	if err != nil {
		slog.Error("OnACLCheck: Invalid topic format", slog.String("topic", topic))
		// invalid topic format.
		return false
	}

	// 3. Agents of messages must include their sender ID
	if write && senderID != loginID {
		slog.Error("OnACLCheck: missing senderID in topic:", slog.String("topic", topic))
		return false
	}

	// 4. roles must be set
	if hook.rolePermissions == nil {
		slog.Error("OnACLCheck: Role permissions not set")
		return false
	}

	// 5. determine the role's permissions
	rolePerm, found := hook.GetRolePermissions(prof.Role, loginID)
	if !found {
		slog.Info("OnACLCheck: Unknown role",
			slog.String("role", prof.Role),
			slog.String("clientID", loginID))
		return false
	}

	// 6. match the role's permissions
	for _, perm := range rolePerm {
		// substitute the clientID in the agentID with the loginID
		permAgentID := perm.AgentID
		if permAgentID == "{clientID}" {
			permAgentID = loginID
		}
		// when write, must allow pub, otherwise must allow sub
		if ((write && perm.AllowPub) || (!write && perm.AllowSub)) &&
			(perm.MsgType == "" || perm.MsgType == msgType) &&
			(perm.AgentID == "" || permAgentID == agentID) &&
			(perm.ThingID == "" || perm.ThingID == thingID) &&
			(perm.MsgName == "" || perm.MsgName == name) {
			if write {
				slog.Debug("OnAclCheck. Publish granted to topic",
					slog.String("clientID", loginID),
					slog.String("topic", topic))
			} else {
				slog.Debug("OnAclCheck. Subscribe granted to topic",
					slog.String("clientID", loginID),
					slog.String("topic", topic))
			}
			return true
		}
	}

	slog.Warn("OnAclCheck. Role doesn't have permissions",
		slog.String("clientID", loginID),
		slog.String("CID", cl.ID),
		slog.String("topic", topic),
		slog.String("role", prof.Role),
		slog.Bool("pub", write))
	return false
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
			MsgType:  transports.MessageTypeRPC,
			AgentID:  serviceID,
			ThingID:  capability,
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
	clientID string, token string, signedNonce string, nonce string) (err error) {

	_, err = jwtauth.ValidateToken(clientID, token, hook.signingKey, signedNonce, nonce)
	//slog.Debug("ValidateToken", "clientID", clientID, "err", err)
	return err
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

	//slog.Info("ValidatePassword", "loginID", loginID, "err", err)

	return cinfo, err
}

func NewMqttAuthHook(signingKey keys.IHiveKey) *MqttAuthHook {
	//slog.Warn("NewMqttAuthHook: ", slog.String("signingKeyPub", signingKeyPubStr))
	hook := &MqttAuthHook{
		HookBase:           mqtt.HookBase{},
		authClients:        nil,
		rolePermissions:    nil,
		authMux:            sync.RWMutex{},
		signingKey:         signingKey,
		servicePermissions: make(map[string][]msgserver.RolePermission),
	}
	return hook
}
