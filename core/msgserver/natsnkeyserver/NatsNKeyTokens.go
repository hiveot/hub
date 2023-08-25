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
)

// CreateKeyPair returns a new user nkey with its public key string
//func (srv *NatsNKeyServer) CreateKeyPair() (interface{}, string) {
//	kp, _ := nkeys.CreateUser()
//	pubKey, _ := kp.PublicKey()
//	return kp, pubKey
//}

// CreateToken uses the public key as token when using nkeys
func (srv *NatsNKeyServer) CreateToken(
	clientID string, clientType string, pubKey string, validitySec int) (token string, err error) {
	if srv.chook == nil {
		return pubKey, nil
	}
	return pubKey, nil
}

// construct a permissions object for a client and its group memberships
// if groupRoles is nil or empty then the user has permissions denied to pub/sub
// to "$JS.API.CONSUMER.>".
func (srv *NatsNKeyServer) makePermissions(
	clientProf *authn.ClientProfile, groupsRole authz.RoleMap) *server.Permissions {

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
	subInbox := "_INBOX." + clientProf.ClientID + ".>"
	subPerm.Allow = append(subPerm.Allow, subInbox)

	// services can pub/sub actions and events
	if clientProf.ClientType == authn.ClientTypeService {
		// publish actions to any thing
		// subscribe events from any thing
		pubService := natshubclient.MakeSubject("", "", "action", ">")
		pubPerm.Allow = append(subPerm.Allow, pubService)
		subService := natshubclient.MakeSubject("", "", "event", ">")
		subPerm.Allow = append(subPerm.Allow, subService)
		// publish events from the service
		// subscribe to actions send to this service
		mySubject := natshubclient.MakeSubject("", clientProf.ClientID, "", ">")
		subPerm.Allow = append(subPerm.Allow, mySubject)
		pubPerm.Allow = append(subPerm.Allow, mySubject)
	} else if clientProf.ClientType == authn.ClientTypeDevice {
		// devices can pub/sub on their own address
		mySubject := natshubclient.MakeSubject(clientProf.ClientID, "", "", ">")
		pubPerm.Allow = append(subPerm.Allow, mySubject)
		subPerm.Allow = append(subPerm.Allow, mySubject)
	} else if clientProf.ClientType == authn.ClientTypeUser {
		// when users have no roles, they cannot use consumers or streams
		if groupsRole == nil || len(groupsRole) == 0 {
			pubPerm.Deny = append(pubPerm.Deny, "$JS.API.CONSUMER.>")
			pubPerm.Deny = append(pubPerm.Deny, "$JS.API.STREAM.>")
		}
		// FIXME: server shouldn't give access to services.
		// how to manage this access?
		subject := natshubclient.MakeSubject(
			authn.AuthnServiceName,
			authn.ClientAuthnCapability,
			natshubclient.SubjectTypeAction, ">")
		pubPerm.Allow = append(pubPerm.Allow, subject)
	}

	// users and services can subscribe to streams (groups) they are a member of.
	if groupsRole != nil &&
		(clientProf.ClientType == authn.ClientTypeUser || clientProf.ClientType == authn.ClientTypeService) {
		for groupName, role := range groupsRole {
			// group members can read from the stream
			// FIXME: is any of this needed?
			subPerm.Allow = append(subPerm.Allow, []string{
				"$JS.API.>",
			}...)
			pubPerm.Allow = append(pubPerm.Allow, []string{
				"$JS.API.>",
				"$JS.API.CONSUMER.CREATE." + groupName,
				"$JS.API.CONSUMER.LIST." + groupName,
				"$JS.API.CONSUMER.INFO." + groupName + ".>",     // to get consumer info?
				"$JS.API.CONSUMER.MSG.NEXT." + groupName + ".>", // to get consumer info?
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
