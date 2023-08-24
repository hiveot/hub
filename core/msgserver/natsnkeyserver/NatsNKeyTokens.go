package natsnkeyserver

import (
	"encoding/base64"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/hubclient/natshubclient"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nkeys"
	"time"
)

// CreateJWTToken for authentication and authorization with NATS server and JetStream.
// FIXME: use the same permissions as for the static server clients
//
// https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/jwt:
// NATS further restricts JWTs by requiring that JWTs be:
//
//   - Digitally signed always and only using Ed25519
//   - All Issuer and Subject fields in a JWT must be a public NKEY
//   - Issuer and Subject must match specific NKey roles
//     NKey Roles are operators, accounts and users,
//     Operator is the issuer of account, account issuer of users.
func (srv *NatsNKeyServer) CreateJWTToken(
	clientID string, clientType string, pubKey string, validitySec int) (token string, err error) {

	// create jwt claims that identifies the user and its permissions
	userClaims := jwt.NewUserClaims(pubKey)
	// the token must be issued by a known account
	userClaims.IssuerAccount, _ = srv.cfg.AppAccountKP.PublicKey()
	// can't use claim ID as it is replaced by a hash by Encode(kp)
	userClaims.Name = clientID
	userClaims.Tags.Add("clientType", clientType)
	userClaims.IssuedAt = time.Now().Unix()
	userClaims.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()

	//permissions := srv.makePermissions(clientProf, groupRole)
	//userClaims.Permissions = permissions

	// everyone can subscribe to actions aimed at them
	userClaims.Permissions.Sub.Allow.Add("things." + clientID + ".*.action.>")
	// everyone can publish events from themselves
	userClaims.Permissions.Pub.Allow.Add("things." + clientID + ".*.event.>")
	// everyone can publish authn client requests
	userClaims.Permissions.Pub.Allow.Add("things." + authn.AuthnServiceName + "." + authn.ClientAuthnCapability + ".action.>")
	// everyone can sub to their inbox (using inbox prefix)
	userClaims.Permissions.Sub.Allow.Add("_INBOX." + clientID + ".>")

	if clientType == authn.ClientTypeDevice {
		//userClaims.Limits.Data = 1 * 1024 * 1024 // max message size???
		// devices can publish to any inbox (to respond to action requests)
		userClaims.Permissions.Pub.Allow.Add("_INBOX.>")
	} else if clientType == authn.ClientTypeService {
		//userClaims.Limits.Data = 1 * 1024 * 1024 // limit of what???
		// services can subscribe to any event
		userClaims.Permissions.Sub.Allow.Add("things.*.*.event.>")
		// services can publish to any inbox to respond to actions
		userClaims.Permissions.Pub.Allow.Add("_INBOX.>")
		// services can publish to manage streams
		userClaims.Permissions.Pub.Allow.Add("$JS.API.STREAM.>")
		//// services can subscribe to groups
		//userClaims.Permissions.Sub.Allow.Add("groups.>")
		//userClaims.Permissions.Pub.Allow.Add("groups.>")
	} else if clientType == authn.ClientTypeUser {
		// users can subscribe to the built-in all group
		//userClaims.Limits.Data = 1 * 1024 * 1024 // max data this client can ... do?
		userClaims.Permissions.Sub.Allow.Add("$JS.API.STREAM.ALL.>")
	} else {
		userClaims.Limits.Subs = 0 // ??? no subscription allowed
		userClaims.Permissions.Pub.Deny.Add(">")
		userClaims.Permissions.Sub.Deny.Add(">")
	}

	// sign the claims with the service signing key
	token, err = userClaims.Encode(srv.cfg.AppAccountKP)
	if err != nil {
		err = fmt.Errorf("failed creating new token: %w", err)
	}
	return token, err
}

// CreateKeyPair returns a new user nkey with its public key string
//func (srv *NatsNKeyServer) CreateKeyPair() (interface{}, string) {
//	kp, _ := nkeys.CreateUser()
//	pubKey, _ := kp.PublicKey()
//	return kp, pubKey
//}

func (srv *NatsNKeyServer) CreateToken(
	clientID string, clientType string, pubKey string, validitySec int) (token string, err error) {
	if srv.chook == nil {
		return pubKey, nil
	}
	return srv.CreateJWTToken(clientID, clientType, pubKey, validitySec)
}

// construct a permissions object for a client and its group memberships
// if groupRoles is nil then nil permissions is returned (eg unlimited)
func (srv *NatsNKeyServer) makePermissions(
	clientProf *authn.ClientProfile, groupsRole authz.RoleMap) *server.Permissions {
	if groupsRole == nil {
		return nil
	}
	subPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  nil,
	}
	pubPerm := server.SubjectPermission{
		Allow: []string{},
		Deny:  nil,
	}
	perm := &server.Permissions{
		Publish:   nil,
		Subscribe: nil,
		Response:  nil,
	}
	// all clients can use their inbox, using inbox prefix
	subInbox := "_INBOX." + clientProf.ClientID + ".>"
	subPerm.Allow = append(subPerm.Allow, subInbox)

	// services can pub actions and subscribe
	if clientProf.ClientType == authn.ClientTypeService {
		pubService := natshubclient.MakeSubject("", "", "action", ">")
		pubPerm.Allow = append(subPerm.Allow, pubService)
		subService := natshubclient.MakeSubject("", "", "event", ">")
		subPerm.Allow = append(subPerm.Allow, subService)
	}
	// users and services can subscribe to streams (groups)
	if clientProf.ClientType == authn.ClientTypeUser || clientProf.ClientType == authn.ClientTypeService {
		for groupName, role := range groupsRole {
			// group members can read from the stream
			// FIXME: is any of this needed?
			subPerm.Allow = append(subPerm.Allow, []string{}...)
			pubPerm.Allow = append(pubPerm.Allow, []string{
				//"$JS.API.CONSUMER.CREATE." + groupName,
				//"$JS.API.CONSUMER.LIST." + groupName,
				//"$JS.API.CONSUMER.INFO." + groupName + ">",     // to get consumer info?
				//"$JS.API.CONSUMER.MSG.NEXT." + groupName + ">", // to get consumer info?
			}...)

			// TODO: operators and managers can publish actions for all things in the group
			// Can we use a stream publish that mapped back to the thing?
			// eg: {groupName}.{publisher}.{thing}.action.>
			// maps to things.{publisher}.{thing}.action.>
			// where the stream has a filter on all things added to the stream?
			if role == authz.GroupRoleOperator || role == authz.GroupRoleManager {
				actionSubj := groupName + ".*.*.action.>"
				pubPerm.Allow = append(pubPerm.Allow, actionSubj)
			}
		}
	}
	if clientProf.ClientType == authn.ClientTypeDevice {
		// devices can pub/sub on their own address and their inbox
		pubDevice := natshubclient.MakeSubject(clientProf.ClientID, "", "event", ">")
		pubPerm.Allow = append(subPerm.Allow, pubDevice)
		subDevice := natshubclient.MakeSubject(clientProf.ClientID, "", "action", ">")
		subPerm.Allow = append(subPerm.Allow, subDevice)
	}
	return perm
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
