package natsnkeyserver

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/auth"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"time"
)

// NatsJWTTokenizer generates and verifies NATS style JWT tokens.
// This implements the IAuthnTokenizer interface
type NatsJWTTokenizer struct {
	accountKey  nkeys.KeyPair
	accountName string
}

// CreatePermissions creates client permissions for embedding in the JWT token
func (svc *NatsJWTTokenizer) CreatePermissions(clientID string, clientType string) jwt.Permissions {
	perm := jwt.Permissions{
		Pub: jwt.Permission{},
		Sub: jwt.Permission{},
	}
	// TODO: can this use the server's authz permission creation?

	// everyone can subscribe to actions aimed at them
	perm.Sub.Allow.Add("things." + clientID + ".*.action.>")
	// everyone can publish events from themselves
	perm.Pub.Allow.Add("things." + clientID + ".*.event.>")
	// everyone can publish their own authn profile requests
	perm.Pub.Allow.Add("things." + auth.AuthServiceName +
		"." + auth.AuthProfileCapability + ".action.*." + clientID)
	// everyone can sub to their inbox (using inbox prefix)
	perm.Sub.Allow.Add("_INBOX." + clientID + ".>")

	if clientType == auth.ClientTypeDevice {
		//userClaims.Limits.Data = 1 * 1024 * 1024 // max message size???
		// devices can publish to any inbox (to respond to action requests)
		perm.Pub.Allow.Add("_INBOX.>")
	} else if clientType == auth.ClientTypeService {
		//userClaims.Limits.Data = 1 * 1024 * 1024 // limit of what???
		// services can subscribe to any event
		perm.Sub.Allow.Add("things.*.*.event.>")
		// services can publish to any inbox to respond to actions
		perm.Pub.Allow.Add("_INBOX.>")
		// services can publish to manage streams and consumers
		perm.Pub.Allow.Add("$JS.API.STREAM.>")
		perm.Pub.Allow.Add("$JS.API.CONSUMER.>")
		// services can subscribe to capability RPC requests
		perm.Sub.Allow.Add("svc." + clientID + ".*.action.>")
	} else if clientType == auth.ClientTypeUser {
		// users can create to the built-in all group
		//userClaims.Limits.Data = 1 * 1024 * 1024 // max data this client can ... do?
		perm.Pub.Allow.Add("$JS.API.CONSUMER." + EventsIntakeStreamName + ".>")
	} else {
		perm.Pub.Deny.Add(">")
		perm.Sub.Deny.Add(">")
	}
	return perm
}

// CreateToken returns a new user jwt token signed by the issuer account.
//
// Note in server mode the issuer account must be the same account as the
// account the client belongs to.
//
//	clientID is the user's login/connect ID which is added as the token ID
//	clientType identifies type of client: device, service, ...
//	clientPub is the users's public key which goes into the subject field of the jwt token
//	validity is the validity period of the token
func (svc *NatsJWTTokenizer) CreateToken(
	clientID string, clientType string, clientPub string, validity time.Duration,
) (newToken string, err error) {

	// build an jwt response; user_nkey (clientPub) is the subject
	uc := jwt.NewUserClaims(clientPub)

	// can't use claim ID as it is replaced by a hash by Encode(kp)
	uc.Name = clientID
	uc.Tags.Add("clientType", clientType)
	uc.IssuedAt = time.Now().Unix()
	uc.Expires = time.Now().Add(validity).Unix()

	uc.Name = clientID
	// Note: In server mode do not set issuer account. This is for operator mode only.
	// Using IssuerAccount in server mode is unnecessary and fails with:
	//   "Error non operator mode account %q: attempted to use issuer_account"
	// not sure why this is an issue...
	//uc.IssuerAccount,_ = svr.calloutAcctKey.PublicKey()
	//uc.Issuer, _ = chook.appAcctKey.PublicKey()

	uc.IssuedAt = time.Now().Unix()

	// Note: in server mode 'aud' should contain the account name. In operator mode it expects
	// the account key.
	// see also: https://github.com/nats-io/nats-server/issues/4313
	//uc.Audience, _ = svc.appAcctKey.PublicKey()
	uc.Audience = svc.accountName

	//uc.UserPermissionLimits = *limits // todo
	uc.Permissions = svc.CreatePermissions(clientID, clientType)

	// check things are valid
	vr := jwt.CreateValidationResults()
	uc.Validate(vr)
	if len(vr.Errors()) != 0 || len(vr.Warnings()) > 0 {
		err = fmt.Errorf("validation error or warning: %w", vr.Errors()[0])
	}
	// encode sets the issuer field to the public key
	newToken, err = uc.Encode(svc.accountKey)
	//newToken, err = uc.Encode(chook.calloutAccountKey)
	return newToken, err
}

// ValidateToken verifies a NATS JWT token
//   - verify if jwtToken is a valid token
//   - validate the token isn't expired
//   - verify the user's public key's nonce based signature
//     this can only be signed when the user has its private key
//   - verify the issuer is the signing/account key.
//
// Verifying the signedNonce is optional. Use "" to ignore.
func (svc *NatsJWTTokenizer) ValidateToken(
	clientID string, pubKey string, tokenString string, signedNonce string, nonce string) error {

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
		err = errors.New(warns[0])
	}
	// the subject contains the public user nkey
	userKey, err := nkeys.FromPublicKey(arc.Subject)
	if err != nil {
		return fmt.Errorf("user nkey not valid: %w", err)
	}
	userPub, _ := userKey.PublicKey()
	if userPub != pubKey {
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
		if err = userKey.Verify([]byte(nonce), sig); err != nil {
			return fmt.Errorf("signature not verified")
		}
	}
	// verify issuer account matches
	accPub, _ := svc.accountKey.PublicKey()
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
	return err
}

// NewNatsJWTTokenizer handles token generation and verification for the NATS messaging server
//
//	accountKey is used to sign JWT tokens
func NewNatsJWTTokenizer(accountName string, accountKey nkeys.KeyPair) *NatsJWTTokenizer {

	tokenizer := &NatsJWTTokenizer{
		accountKey:  accountKey,
		accountName: accountName,
	}
	return tokenizer
}