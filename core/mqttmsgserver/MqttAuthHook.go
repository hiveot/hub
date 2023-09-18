package mqttmsgserver

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/hubcl/mqtthubclient"
	mqtt "github.com/mochi-mqtt/server/v2"
	"github.com/mochi-mqtt/server/v2/packets"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/exp/slog"
	"strings"
	"sync"
	"time"
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

	if authInfo.ClientID == "" || authInfo.ClientType == "" {
		err = fmt.Errorf("CreateToken: Missing client ID or type")
		return "", err
	} else if authInfo.PubKey == "" {
		err = fmt.Errorf("CreateToken: client has no public key")
		return "", err
	} else if authInfo.Role == "" {
		err = fmt.Errorf("CreateToken: client has no role")
		return "", err
	}

	// see also: https://golang-jwt.github.io/jwt/usage/create/
	// TBD: use validity period from profile
	// default validity period depends on client type (why?)
	validity := auth.DefaultUserTokenValidityDays
	if authInfo.ClientType == auth.ClientTypeDevice {
		validity = auth.DefaultDeviceTokenValidityDays
	} else if authInfo.ClientType == auth.ClientTypeService {
		validity = auth.DefaultServiceTokenValidityDays
	}
	expiryTime := time.Now().Add(time.Duration(validity) * time.Hour * 24)
	// Create the JWT claims, which includes the username, clientType and expiry time
	claims := jwt.MapClaims{
		//"alg": "ES256", // jwt.SigningMethodES256,
		"typ": "JWT",
		"aud": authInfo.ClientType, //
		"sub": authInfo.PubKey,     // public key of client (same as nats)
		"iss": hook.signingKeyPub,
		"exp": expiryTime.Unix(), // expiry time. Seconds since epoch
		"iat": time.Now().Unix(), // issued at. Seconds since epoch

		// custom claim fields
		"clientID":   authInfo.ClientID,
		"clientType": authInfo.ClientType,
		"role":       authInfo.Role,
	}

	// Declare the token with the algorithm used for signing, and the claims
	claimsToken := jwt.NewWithClaims(jwt.SigningMethodES256, claims)
	authToken, err := claimsToken.SignedString(hook.signingKey)
	if err != nil {
		return "", err
	}

	return authToken, nil
}

// GetClientAuth returns the client auth info for the given ID
// This returns an error if the client is not found
func (hook *MqttAuthHook) GetClientAuth(clientID string) (msgserver.ClientAuthInfo, error) {
	hook.authMux.RLock()
	clientAuth, found := hook.authClients[clientID]
	hook.authMux.RUnlock()
	if !found {
		return clientAuth, fmt.Errorf("client %s not known", clientID)
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

	// verify authentication using password or token
	// step 1: credentials must be provided
	if pk.Connect.PasswordFlag == false || len(pk.Connect.Password) == 0 {
		slog.Info("OnConnectAuthenticate: missing authn credentials",
			slog.String("clientID", clientID),
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

	// step 3: password authentication, the user must be known
	authInfo, found := hook.authClients[clientID]
	if !found {
		slog.Info("OnConnectAuthenticate: unknown user",
			slog.String("clientID", clientID),
			slog.String("cid", cid))
		return false
	}
	if authInfo.PasswordHash != "" {
		// verify password
		err := bcrypt.CompareHashAndPassword([]byte(authInfo.PasswordHash), pk.Connect.Password)
		if err != nil {
			slog.Info("OnConnectAuthenticate: invalid password",
				"clientID", clientID,
				"net.remote", cl.Net.Remote)
			return false
		}
		slog.Info("OnConnectAuthenticate: password login success",
			"clientID", clientID,
			"net.remote", cl.Net.Remote)
		return true
	}
	// credentials provided but unable to match it
	slog.Info("OnConnectAuthenticate: invalid credentials",
		"clientID", clientID,
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

	prefix, deviceID, thingID, stype, name, senderID, err :=
		mqtthubclient.SplitTopic(topic)
	if err != nil {
		// invalid topic format.
		return false
	}
	_ = prefix
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
			(perm.MsgType == "" || perm.MsgType == stype) &&
			(perm.SourceID == "" || perm.SourceID == deviceID) &&
			(perm.ThingID == "" || perm.ThingID == thingID) &&
			(perm.MsgName == "" || perm.MsgName == name) &&
			(perm.Prefix == "" || perm.Prefix == prefix) {
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

	claims := jwt.MapClaims{}
	jwtToken, err := jwt.ParseWithClaims(token, &claims,
		func(token *jwt.Token) (interface{}, error) {
			return &hook.signingKey.PublicKey, nil
		}, jwt.WithValidMethods([]string{
			jwt.SigningMethodES256.Name,
			jwt.SigningMethodES384.Name,
			jwt.SigningMethodES512.Name,
			"EdDSA",
		}),
		jwt.WithIssuer(hook.signingKeyPub), // url encoded string
	)
	if err != nil || jwtToken == nil || !jwtToken.Valid {
		return authInfo, fmt.Errorf("invalid JWT token: %s", err)
	}

	pubKey, _ := claims.GetSubject()
	authInfo.PubKey = pubKey
	if pubKey == "" {
		return authInfo, fmt.Errorf("token has no public key")
	}

	jwtClientType, _ := claims["clientType"]
	authInfo.ClientType = jwtClientType.(string)
	if authInfo.ClientType == "" {
		return authInfo, fmt.Errorf("token has no client type")
	}

	authInfo.ClientID = clientID
	jwtClientID, _ := claims["clientID"]
	if jwtClientID != clientID {
		// while this doesn't provide much extra security it might help
		// prevent bugs. Potentially also useful as second factor auth check if
		// clientID is obtained through a different means.
		return authInfo, fmt.Errorf("token belongs to different clientID")
	}

	jwtRole, _ := claims["role"]
	authInfo.Role = jwtRole.(string)
	if authInfo.Role == "" {
		return authInfo, fmt.Errorf("token has no role")
	}

	return authInfo, nil
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
