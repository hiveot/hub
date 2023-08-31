package natsnkeyserver

import (
	"encoding/base64"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/slog"
	"time"
)

// viewers can subscribe to all things
var viewerPermissions = []msgserver.RolePermission{{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeEvent,
	AllowSub: true,
}}

// operators can also publish thing actions
var operatorPermissions = append(viewerPermissions, msgserver.RolePermission{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeAction,
	AllowPub: true,
})

// managers can also publish configuration
var managerPermissions = append(operatorPermissions, msgserver.RolePermission{
	Prefix:   "things",
	MsgType:  natshubclient.MessageTypeConfig,
	AllowPub: true,
})

// administrators can do all and publish to services
var adminPermissions = append(managerPermissions, msgserver.RolePermission{
	Prefix:   "svc",
	MsgType:  natshubclient.MessageTypeAction,
	AllowPub: true,
	AllowSub: true,
})

// DefaultRolePermissions contains the default pub/sub permissions for each user role
var DefaultRolePermissions = map[string][]msgserver.RolePermission{
	auth.ClientRoleViewer:   viewerPermissions,
	auth.ClientRoleOperator: operatorPermissions,
	auth.ClientRoleManager:  managerPermissions,
	auth.ClientRoleAdmin:    adminPermissions,
	auth.ClientRoleNone:     nil,
}

// ServicePermissions defines for each role the service capability that can be used
var ServicePermissions = map[string][]msgserver.RolePermission{}

// ApplyAuth reconfigures the server for authentication and authorization.
// For each client this applies the permissions associated with the client type and role.
//
//	Role permissions can be changed with 'SetRolePermissions'.
//	Service permissions can be set with 'SetServicePermissions'
func (srv *NatsNKeyServer) ApplyAuth(clients []msgserver.AuthClient) error {

	// password users authenticate with password while nkey users authenticate with key-pairs.
	// clients can use both.
	pwUsers := []*server.User{}
	nkeyUsers := []*server.NkeyUser{}

	// keep the core service that was added on server start
	coreServicePub, _ := srv.cfg.CoreServiceKP.PublicKey()
	nkeyUsers = append(nkeyUsers, &server.NkeyUser{
		Nkey:        coreServicePub,
		Permissions: nil, // unlimited access
		Account:     srv.cfg.appAcct,
	})

	// apply authn all clients
	for _, clientInfo := range clients {
		userPermissions := srv.MakePermissions(clientInfo, srv.rolePermissions)

		if clientInfo.PasswordHash != "" {
			pwUsers = append(pwUsers, &server.User{
				Username:    clientInfo.ClientID,
				Password:    clientInfo.PasswordHash,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
			})
		}

		if clientInfo.PubKey != "" {
			// add an nkey entry
			nkeyUsers = append(nkeyUsers, &server.NkeyUser{
				Nkey:        clientInfo.PubKey,
				Permissions: userPermissions,
				Account:     srv.cfg.appAcct,
			})
		}
	}
	srv.natsOpts.Users = pwUsers
	srv.natsOpts.Nkeys = nkeyUsers

	err := srv.ns.ReloadOptions(&srv.natsOpts)
	return err
}

// CreateToken uses the public key as token when using nkeys
func (srv *NatsNKeyServer) CreateToken(
	clientID string, clientType string, pubKey string, tokenValidity time.Duration) (token string, err error) {
	if srv.chook == nil {
		return pubKey, nil
	}
	return pubKey, nil
}

// MakePermissions constructs a permissions object for a client
// Clients that are sources (device,service) receive hard-coded permissions, while users (user,service) permissions
// are based on their role.
func (srv *NatsNKeyServer) MakePermissions(
	clientInfo msgserver.AuthClient,
	authzRoles map[string][]msgserver.RolePermission) *server.Permissions {

	subPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  []string{},
	}
	pubPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  []string{},
	}
	perm := &server.Permissions{
		Publish:   &pubPerm,
		Subscribe: &subPerm,
		Response:  nil,
	}
	// all clients can use their inbox, using inbox prefix
	subInbox := "_INBOX." + clientInfo.ClientID + ".>"
	subPerm.Allow = append(subPerm.Allow, subInbox)

	rolePerm, found := authzRoles[clientInfo.Role]
	if found && rolePerm == nil {
		// no permissions for this role
	} else if found {
		// apply role permissions
		for _, rp := range rolePerm {
			subj := natshubclient.MakeThingsSubject(rp.SourceID, rp.ThingID, rp.MsgType, rp.MsgName)
			if rp.AllowPub {
				pubPerm.Allow = append(pubPerm.Allow, subj)
			}
			if rp.AllowSub {
				subPerm.Allow = append(subPerm.Allow, subj)
			}
		}
		// allow event stream access
		streamName := EventsIntakeStreamName
		pubPerm.Allow = append(pubPerm.Allow, []string{
			//"$JS.API.>", // FIXME: remove after things start to work
			"$JS.API.CONSUMER.CREATE." + streamName,
			"$JS.API.CONSUMER.LIST." + streamName,
			"$JS.API.CONSUMER.INFO." + streamName + ".>",     // to get consumer info?
			"$JS.API.CONSUMER.MSG.NEXT." + streamName + ".>", // to get consumer info?
		}...)
	} else {
		// unknown role
		slog.Error("unknown role",
			"clientID", clientInfo.ClientID, "clientType", clientInfo.ClientType,
			"role", clientInfo.Role)

		// when clients have no roles, they cannot use consumers or streams
		pubPerm.Deny = append(pubPerm.Deny, "$JS.API.CONSUMER.>")
		pubPerm.Deny = append(pubPerm.Deny, "$JS.API.STREAM.>")
	}

	// user role might be allowed to use a service capability
	sp, found := ServicePermissions[clientInfo.Role]
	if found && sp != nil {
		for _, perm := range sp {
			subject := natshubclient.MakeServiceSubject(
				perm.SourceID, perm.ThingID, natshubclient.MessageTypeAction, "")
			pubPerm.Allow = append(pubPerm.Allow, subject)
		}
	}

	// devices and services are sources that can publish events and subscribe to actions and config requests
	if clientInfo.ClientType == auth.ClientTypeDevice || clientInfo.ClientType == auth.ClientTypeService {
		subject1 := natshubclient.MakeThingsSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeEvent, "")
		subject2 := natshubclient.MakeThingsSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeAction, "")
		subject3 := natshubclient.MakeThingsSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeConfig, "")
		pubPerm.Allow = append(pubPerm.Allow, subject1)
		subPerm.Allow = append(subPerm.Allow, subject2, subject3)
	}

	// services can also subscribe to actions on the svc prefix
	if clientInfo.ClientType == auth.ClientTypeService {
		subject1 := natshubclient.MakeServiceSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeEvent, "")
		subject2 := natshubclient.MakeServiceSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeAction, "")
		subject3 := natshubclient.MakeServiceSubject(
			clientInfo.ClientID, "", natshubclient.MessageTypeConfig, "")
		pubPerm.Allow = append(pubPerm.Allow, subject1)
		subPerm.Allow = append(subPerm.Allow, subject2, subject3)
	}

	return perm
}

// SetRolePermissions sets a custom map of user role->[]permissions
func (srv *NatsNKeyServer) SetRolePermissions(
	rolePerms map[string][]msgserver.RolePermission) {
	srv.rolePermissions = rolePerms
}

// SetServicePermissions adds the service permissions to the roles
func (srv *NatsNKeyServer) SetServicePermissions(
	serviceID string, capability string, roles []string) {

	for _, role := range roles {
		// add the role if needed
		rp := ServicePermissions[role]
		if rp == nil {
			rp = []msgserver.RolePermission{}
		}
		rp = append(rp, msgserver.RolePermission{
			Prefix:   "svc",
			SourceID: serviceID,
			ThingID:  capability,
			MsgType:  natshubclient.MessageTypeAction,
			MsgName:  "", // all methods of the capability can be used
			AllowPub: true,
			AllowSub: false,
		})
		ServicePermissions[role] = rp
	}

}

// ValidateJWTToken checks if the given token belongs the clientID and is valid.
//   - verify if jwtToken is a valid token
//   - validate the token isn't expired
//   - verify the user's public key's nonce based signature
//     this can only be signed when the user has its private key
//   - verify the issuer is the signing/account key.
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (srv *NatsNKeyServer) ValidateJWTToken(
	clientID string, pubKey string, jwtToken string, signedNonce string, nonce string) (err error) {

	// the jwt token is not in the JWT field. Workaround by storing it in the token field.
	juc, err := jwt.DecodeUserClaims(jwtToken)
	if err != nil {
		return fmt.Errorf("unable to decode jwt token:%w", err)
	}
	// validate the jwt user claims (not expired)
	vr := jwt.CreateValidationResults()
	juc.Validate(vr)
	if len(vr.Errors()) > 0 {
		return fmt.Errorf("jwt authn failed: %w", vr.Errors()[0])
	}
	// the subject contains the public user nkey
	userPub, err := nkeys.FromPublicKey(juc.Subject)
	if err != nil {
		return fmt.Errorf("user nkey not valid: %w", err)
	}

	// Verify the nonce based token signature
	if signedNonce != "" {
		sig, err := base64.RawURLEncoding.DecodeString(signedNonce)
		if err != nil {
			// Allow fallback to normal base64.
			sig, err = base64.StdEncoding.DecodeString(signedNonce)
			if err != nil {
				return fmt.Errorf("signature not valid base64: %w", err)
			}
		}
		// verify the signature of the public key using the nonce
		// this tells us the user public key is not forged
		if err = userPub.Verify([]byte(nonce), sig); err != nil {
			return fmt.Errorf("signature not verified")
		}
	}
	// verify issuer account matches
	accPub, _ := srv.cfg.AppAccountKP.PublicKey()
	if juc.Issuer != accPub {
		return fmt.Errorf("JWT issuer is not known")
	}
	// clientID must match the user
	if juc.Name != clientID {
		return fmt.Errorf("clientID doesn't match user")
	}

	//if entry.PubKey != userPub {
	//	return fmt.Errorf("user %s public key mismatch", clientID)
	//}

	//acc, err := svc.ns.LookupAccount(juc.IssuerAccount)
	//if err != nil {
	//	return fmt.Errorf("JWT issuer is not known")
	//}
	//if acc.IsExpired() {
	//	return fmt.Errorf("Account JWT has expired")
	//}
	// no access to account revocation list
	//if acc.checkUserRevoked(juc.Subject, juc.IssuedAt) {
	//	return fmt.Errorf("User authentication revoked")
	//}

	//if !validateSrc(juc, c.host) {
	//	return fmt.Errorf("Bad src Ip %s", c.host)
	//	return false
	//}
	return nil
}

// ValidateToken checks if the given token belongs the clientID and is valid.
// When keys is used this returns success
// When nkeys is not used this validates the JWT token
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (srv *NatsNKeyServer) ValidateToken(
	clientID string, pubKey string, oldToken string, signedNonce string, nonce string) (err error) {
	if srv.chook == nil {
		// nkeys only
		if oldToken == "" || pubKey != oldToken {
			return fmt.Errorf("invalid old token for client '%s'", clientID)
		}
		return nil
	}
	return srv.ValidateJWTToken(clientID, pubKey, oldToken, signedNonce, nonce)
}
