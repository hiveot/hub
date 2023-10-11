package service

import (
	"encoding/base64"
	"fmt"
	auth2 "github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/core/msgserver"
	"github.com/hiveot/hub/lib/hubclient/natshubclient"
	"github.com/hiveot/hub/lib/vocab"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
	"golang.org/x/crypto/bcrypt"
	"log/slog"
	"time"
)

// ServicePermissions defines for each role the service capability that can be used
//var ServicePermissions = map[string][]msgserver.RolePermission{}

// ApplyAuth reconfigures the server for authentication and authorization.
// For each client this applies the permissions associated with the client type and role.
//
//	Role permissions can be changed with 'SetRolePermissions'.
//	Service permissions can be set with 'SetServicePermissions'
func (srv *NatsMsgServer) ApplyAuth(clients []msgserver.ClientAuthInfo) error {

	// password users authenticate with password while nkey users authenticate with key-pairs.
	// clients can use both.
	pwUsers := []*server.User{}
	nkeyUsers := []*server.NkeyUser{}
	authClients := map[string]msgserver.ClientAuthInfo{}

	// keep the core, admin and system users
	coreServicePub, _ := srv.Config.CoreServiceKP.PublicKey()
	//adminUserPub, _ := srv.Config.AdminUserKP.PublicKey()
	systemUserPub, _ := srv.Config.SystemUserKP.PublicKey()
	nkeyUsers = append(nkeyUsers, []*server.NkeyUser{
		//{
		//Nkey: adminUserPub,
		//Permissions: nil, // unlimited access
		//Account:     srv.Config.AppAcct,
		//},
		{Nkey: coreServicePub,
			Permissions: nil, // unlimited access
			Account:     srv.Config.AppAcct,
		}, {
			Nkey:        systemUserPub,
			Permissions: nil, // unlimited access
			Account:     srv.ns.SystemAccount(),
		},
	}...)

	// apply authn all clients
	// FIXME: when using callouts, don't apply users and have the callout verifier handle them.
	for _, clientInfo := range clients {
		authClients[clientInfo.ClientID] = clientInfo
		userPermissions := srv.MakePermissions(clientInfo)

		if clientInfo.PasswordHash != "" {
			pwUsers = append(pwUsers, &server.User{
				Username:    clientInfo.ClientID,
				Password:    clientInfo.PasswordHash,
				Permissions: userPermissions,
				Account:     srv.Config.AppAcct,
			})
		}

		if clientInfo.PubKey != "" {
			// add an nkey entry
			nkeyUsers = append(nkeyUsers, &server.NkeyUser{
				Nkey:        clientInfo.PubKey,
				Permissions: userPermissions,
				Account:     srv.Config.AppAcct,
			})
		}
	}
	srv.NatsOpts.Users = pwUsers
	srv.NatsOpts.Nkeys = nkeyUsers
	srv.authClients = authClients

	err := srv.ns.ReloadOptions(&srv.NatsOpts)
	return err
}

// CreateKP creates a keypair for use in connecting or signing.
// This returns the key pair and its public key string.
func (srv *NatsMsgServer) CreateKP() (interface{}, string) {
	kp, _ := nkeys.CreateUser()
	pubStr, _ := kp.PublicKey()
	return kp, pubStr
}

// CreateToken create a new authentication token for a client
// In NKey mode this returns the public key.
// In Callout mode this returns a JWT token with permissions.
func (srv *NatsMsgServer) CreateToken(authInfo msgserver.ClientAuthInfo) (token string, err error) {
	//
	if srv.NatsOpts.AuthCallout != nil {
		token, err = srv.CreateJWTToken(authInfo)
	} else {
		// not using callout so use the given public key as token
		token = authInfo.PubKey
		if authInfo.PubKey == "" {
			err = fmt.Errorf("CreateToken requires a public key for client '%s'", authInfo.ClientID)
		}
	}
	return token, err
}

// CreateJWTToken returns a new user jwt token signed by the issuer account.
//
// Note1 in server mode the issuer account must be the same account as that of the
// callout client. i.e.: callout cannot issue a token for a different account.
// Note2 in callout the generated JWT must contain the on-the-fly generated public key for some reason, not he user's public key
//
//	clientID is the user's login/connect ID which is added as the token ID
//	pubKey is the users's public key which goes into the subject field of the jwt token, use "" for client on record
func (srv *NatsMsgServer) CreateJWTToken(authInfo msgserver.ClientAuthInfo) (newToken string, err error) {
	if authInfo.ClientID == "" || authInfo.PubKey == "" ||
		authInfo.Role == "" || authInfo.ClientType == "" {
		return "", fmt.Errorf("invalid auth info")
	}
	// TODO: use validity period from profile
	validity := auth2.DefaultUserTokenValidityDays
	if authInfo.ClientType == auth2.ClientTypeDevice {
		validity = auth2.DefaultDeviceTokenValidityDays
	} else if authInfo.ClientType == auth2.ClientTypeService {
		validity = auth2.DefaultServiceTokenValidityDays
	}

	// build a jwt response; user_nkey (clientPub) is the subject
	uc := jwt.NewUserClaims(authInfo.PubKey)

	// can't use claim ID as it is replaced by a hash by Encode(kp)
	uc.Name = authInfo.ClientID
	uc.Tags.Add("clientType", authInfo.ClientType)
	uc.IssuedAt = time.Now().Unix()
	uc.Expires = time.Now().Add(time.Duration(validity) * time.Hour * 24).Unix()

	// Note: In server mode do not set issuer account. This is for operator mode only.
	// Using IssuerAccount in server mode is unnecessary and fails with:
	//   "Error non operator mode account %q: attempted to use issuer_account"
	// not sure why this is an issue...
	//uc.IssuerAccount,_ = svr.calloutAcctKey.PublicKey()
	uc.Issuer, _ = srv.Config.AppAccountKP.PublicKey()

	uc.IssuedAt = time.Now().Unix()

	// Note: in server mode 'aud' should contain the account name. In operator mode it expects
	// the account key.
	// see also: https://github.com/nats-io/nats-server/issues/4313
	//uc.Audience, _ = srv.appAcctKey.PublicKey()
	uc.Audience = srv.Config.AppAccountName

	//uc.UserPermissionLimits = *limits // todo

	uc.Permissions = srv.MakeJWTPermissions(authInfo)

	// check things are valid
	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) != 0 || len(vr.Warnings()) > 0 {
		err = fmt.Errorf("validation error or warning: %w", vr.Errors()[0])
	}
	// encode sets the issuer field to the public key
	newToken, err = uc.Encode(srv.Config.AppAccountKP)
	//newToken, err = uc.Encode(chook.calloutAccountKey)
	return newToken, err
}

// GetClientAuth returns the client auth info for the given ID
func (srv *NatsMsgServer) GetClientAuth(clientID string) (msgserver.ClientAuthInfo, error) {
	clientAuth, found := srv.authClients[clientID]
	if !found {
		return clientAuth, fmt.Errorf("client %s not known", clientID)
	}
	return clientAuth, nil
}

// MakeJWTPermissions constructs a permissions object for use in a JWT token.
// Nats calllout doesn't use the nats server permissions so convert it to JWT perm.
func (srv *NatsMsgServer) MakeJWTPermissions(clientInfo msgserver.ClientAuthInfo) jwt.Permissions {

	jperm := jwt.Permissions{
		Pub: jwt.Permission{},
		Sub: jwt.Permission{},
	}
	srvperm := srv.MakePermissions(clientInfo)
	jperm.Pub.Allow = srvperm.Publish.Allow
	jperm.Pub.Deny = srvperm.Publish.Deny
	jperm.Sub.Allow = srvperm.Subscribe.Allow
	jperm.Sub.Deny = srvperm.Subscribe.Deny

	return jperm
}

// MakePermissions constructs a permissions object for a client
//
// Clients that are sources (device,service) receive hard-coded permissions, while users (user,service) permissions
// are based on their role.
func (srv *NatsMsgServer) MakePermissions(clientInfo msgserver.ClientAuthInfo) *server.Permissions {

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
	// all clients can subscribe to their own inbox and publish to other inboxes
	subInbox := vocab.MessageTypeINBOX + "." + clientInfo.ClientID + ".>"
	subPerm.Allow = append(subPerm.Allow, subInbox)
	pubInbox := vocab.MessageTypeINBOX + ".>"
	pubPerm.Allow = append(pubPerm.Allow, pubInbox)

	// generate subjects based on role permissions
	rolePerm, found := srv.rolePermissions[clientInfo.Role]
	if found && rolePerm == nil {
		// no permissions for this role
	} else if found {
		// add services role permissions set by services
		servicePerms, found := srv.servicePermissions[clientInfo.Role]
		if found {
			rolePerm = append(rolePerm, servicePerms...)
		}

		// apply role permissions
		for _, perm := range rolePerm {
			// substitute the clientID in the agentID with the loginID
			permAgentID := perm.AgentID
			if permAgentID == "{clientID}" {
				permAgentID = clientInfo.ClientID
			}
			if perm.AllowPub {
				// publishing requires including their own clientID
				pubSubj := natshubclient.MakeSubject(
					perm.MsgType, permAgentID, perm.ThingID, perm.MsgName, clientInfo.ClientID)
				pubPerm.Allow = append(pubPerm.Allow, pubSubj)
			}
			if perm.AllowSub {
				subSubj := natshubclient.MakeSubject(
					perm.MsgType, permAgentID, perm.ThingID, perm.MsgName, "")
				subPerm.Allow = append(subPerm.Allow, subSubj)
			}
		}
		// allow event stream access
		streamName := EventsIntakeStreamName
		pubPerm.Allow = append(pubPerm.Allow, []string{
			"$JS.API.STREAM.INFO." + streamName,
			"$JS.API.CONSUMER.CREATE." + streamName,
			"$JS.API.CONSUMER.LIST." + streamName,
			"$JS.API.CONSUMER.INFO." + streamName + ".>",     // to get consumer info?
			"$JS.API.CONSUMER.MSG.NEXT." + streamName + ".>", // to get consumer info?
		}...)
		// admin role can access all JS API INFO
		if clientInfo.Role == auth2.ClientRoleAdmin {
			pubPerm.Allow = append(pubPerm.Allow, "$JS.API.INFO")
		}
	} else {
		// unknown role
		slog.Error("unknown role",
			"clientID", clientInfo.ClientID, "clientType", clientInfo.ClientType,
			"role", clientInfo.Role)

		// when clients have no roles, they cannot use consumers or streams
		pubPerm.Deny = append(pubPerm.Deny, "$JS.API.CONSUMER.>")
		pubPerm.Deny = append(pubPerm.Deny, "$JS.API.STREAM.>")
	}
	return perm
}

// SetRolePermissions sets a custom map of user role->[]permissions
func (srv *NatsMsgServer) SetRolePermissions(
	rolePerms map[string][]msgserver.RolePermission) {
	srv.rolePermissions = rolePerms
}

// SetServicePermissions adds the service permissions to the roles
func (srv *NatsMsgServer) SetServicePermissions(
	serviceID string, capability string, roles []string) {

	for _, role := range roles {
		// add the role if needed
		rp := srv.servicePermissions[role]
		if rp == nil {
			rp = []msgserver.RolePermission{}
		}
		rp = append(rp, msgserver.RolePermission{
			MsgType:  vocab.MessageTypeRPC,
			AgentID:  serviceID,
			ThingID:  capability,
			MsgName:  "", // all methods of the capability can be used
			AllowPub: true,
			AllowSub: false,
		})
		srv.servicePermissions[role] = rp
	}

}

// ValidateJWTToken verifies a NATS JWT token
//   - verify if jwtToken is a valid token
//   - validate the token isn't expired
//   - verify the user's public key's nonce based signature
//     this can only be signed when the user has its private key
//   - verify the issuer is the signing/account key.
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (srv *NatsMsgServer) ValidateJWTToken(
	clientID string, tokenString string, signedNonce string, nonce string) error {

	arc, err := jwt.DecodeUserClaims(tokenString)
	if err != nil {
		return fmt.Errorf("unable to decode jwt token:%w", err)
	}
	vr := jwt.CreateValidationResults()
	arc.Validate(vr)
	errs := vr.Errors()
	warns := vr.Warnings()
	if len(errs) > 0 {
		err = fmt.Errorf("jwt authn failed: %w", vr.Errors()[0])
	} else if len(warns) > 0 {
		err = fmt.Errorf("jwt auth failed: %s", warns[0])
	}
	// the subject contains the public user nkey
	// TBD: does requiring the client to be on file improve security?
	userAuth, err := srv.GetClientAuth(clientID)
	if err != nil || arc.Subject != userAuth.PubKey {
		return fmt.Errorf("user public key on file doesn't match token")
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
		//userKey, err := nkeys.FromPublicKey(userAuth.PubKey)
		userKey, err := nkeys.FromPublicKey(arc.Subject)
		if err = userKey.Verify([]byte(nonce), sig); err != nil {
			return fmt.Errorf("signature not verified")
		}
	}
	// verify issuer account matches
	accPub, _ := srv.Config.AppAccountKP.PublicKey()
	if arc.Issuer != accPub {
		return fmt.Errorf("JWT issuer is not known")
	}
	// clientID must match the user
	if arc.Name != clientID {
		return fmt.Errorf("clientID doesn't match user")
	}

	//if entry.PubKey != userPub {
	//	return fmt.Errorf("user %s public key mismatch", clientID)
	//}

	//acc, err := srv.ns.LookupAccount(juc.IssuerAccount)
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
	return err
}

// ValidatePassword checks if the given password matches the user
func (srv *NatsMsgServer) ValidatePassword(loginID string, password string) error {
	if loginID == "" || password == "" {
		return fmt.Errorf("ValidatePassword: failed for user '%s'", loginID)
	}
	cAuth, err := srv.GetClientAuth(loginID)
	if err == nil {
		err = bcrypt.CompareHashAndPassword([]byte(cAuth.PasswordHash), []byte(password))
	}
	return err
}

// ValidateNKey checks if the given nkey and nounce belongs the clientID and is valid.
// Intended for use by callout to verify nkey with nonce.
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (srv *NatsMsgServer) ValidateNKey(
	clientID string, pubKey string, signedNonce string, nonce string) (err error) {

	sig, err := base64.RawURLEncoding.DecodeString(signedNonce)
	pub, err := nkeys.FromPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("ValidateNKey: user nkey not valid: %v", err)
	}
	if err := pub.Verify([]byte(nonce), sig); err != nil {
		// invalid signature
		return err
	}

	prof, err := srv.GetClientAuth(clientID)
	if err != nil {
		return err
	}
	if prof.PubKey != pubKey {
		return fmt.Errorf("ValidateNKey: public key not on file")
	}
	return nil
}

// ValidateToken checks if the given token belongs the clientID and is valid.
// When keys is used this returns success
// When nkeys is not used this validates the JWT token
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (srv *NatsMsgServer) ValidateToken(
	clientID string, token string, signedNonce string, nonce string) (err error) {
	if srv.NatsOpts.AuthCallout == nil {
		// nkeys only
		if token == "" || clientID == "" {
			return fmt.Errorf("invalid token for client '%s'", clientID)
		}
		return nil
	}
	return srv.ValidateJWTToken(clientID, token, signedNonce, nonce)
}
