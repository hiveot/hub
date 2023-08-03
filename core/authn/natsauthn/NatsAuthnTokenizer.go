package natsauthn

import (
	"encoding/base64"
	"fmt"
	"github.com/hiveot/hub/api/go/authn"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"time"
)

// NatsAuthnTokenizer generates and validates NATS tokens.
// This implements the IAuthnTokenizer interface
type NatsAuthnTokenizer struct {
	signingKP nkeys.KeyPair
}

// CreateToken for authentication and authorization with NATS server and JetStream
// TODO: determine what the Limits value does
//
// https://docs.nats.io/running-a-nats-service/configuration/securing_nats/auth_intro/jwt:
// NATS further restricts JWTs by requiring that JWTs be:
//
//   - Digitally signed always and only using Ed25519
//   - All Issuer and Subject fields in a JWT must be a public NKEY
//   - Issuer and Subject must match specific NKey roles
//     NKey Roles are operators, accounts and users,
//     Operator is the issuer of account, account issuer of users.
func (svc *NatsAuthnTokenizer) CreateToken(
	clientID string, clientType string, pubKey string, validitySec int) (token string, err error) {

	// create jwt claims that identifies the user and its permissions
	userClaims := jwt.NewUserClaims(pubKey)
	// can't use claim ID as it is replaced by a hash by Encode(kp)
	userClaims.Name = clientID
	userClaims.Tags.Add("clientType", clientType)
	userClaims.IssuedAt = time.Now().Unix()
	userClaims.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()
	if clientType == authn.ClientTypeDevice {
		// devices can subscribe to actions aimed at them
		// devices can publish events of which they are the publisher
		// devices can receive replies in their inbox
		//userClaims.Limits.Data = 1 * 1024 * 1024 // max message size???
		userClaims.Permissions.Sub.Allow.Add("things." + clientID + ".*.action.>")
		userClaims.Permissions.Pub.Allow.Add("things." + clientID + ".*.event.>")
		userClaims.Permissions.Pub.Allow.Add("_INBOX.>")
	} else if clientType == authn.ClientTypeService {
		// services can subscribe to any event and to actions aimed at them
		// services can publish events of which they are the publisher
		//userClaims.Limits.Data = 1 * 1024 * 1024 // limit of what???
		userClaims.Permissions.Sub.Allow.Add("things.*.*.event.>")
		userClaims.Permissions.Sub.Allow.Add("things." + clientID + ".*.action.>")
		userClaims.Permissions.Pub.Allow.Add("_INBOX.>")
	} else if clientType == authn.ClientTypeUser {
		// users can publish actions and subscribe to group events
		//userClaims.Limits.Data = 1 * 1024 * 1024 // max data this client can ... do?
		userClaims.Permissions.Pub.Allow.Add("groups.*.*.action.>")
		userClaims.Permissions.Sub.Allow.Add("groups.*.*.event.>")
		userClaims.Permissions.Sub.Allow.Add("_INBOX.>")
	} else {
		userClaims.Limits.Subs = 0 // ??? no subscription allowed
		userClaims.Permissions.Pub.Deny.Add(">")
		userClaims.Permissions.Sub.Deny.Add(">")
	}

	// sign the claims with the service signing key
	token, err = userClaims.Encode(svc.signingKP)
	return token, err
}

// ValidateToken checks if the given token belongs the clientID and is valid.
//   - verify if jwtToken is a valid token
//   - validate the token isn't expired
//   - verify the user's public key's nonce based signature
//     this can only be signed when the user has its private key
//   - verify the issuer is the signing/account key.
func (svc *NatsAuthnTokenizer) ValidateToken(
	clientID string, jwtToken string, signedNonce string, nonce string) (err error) {

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

	// Verify the nonce based token signature
	sig, err := base64.RawURLEncoding.DecodeString(signedNonce)
	if err != nil {
		// Allow fallback to normal base64.
		sig, err = base64.StdEncoding.DecodeString(signedNonce)
		if err != nil {
			return fmt.Errorf("signature not valid base64: %w", err)
		}
	}
	// the subject contains the public user nkey
	userPub, err := nkeys.FromPublicKey(juc.Subject)
	if err != nil {
		return fmt.Errorf("user nkey not valid: %w", err)
	}
	// verify the signature of the public key using the nonce
	// this tells us the user public key is not forged
	if err = userPub.Verify([]byte(nonce), sig); err != nil {
		return fmt.Errorf("signature not verified")
	}
	// verify issuer account matches
	accPub, _ := svc.signingKP.PublicKey()
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

// NewNatsTokenizer handles token generation and verification for the NATS messaging server
// Call 'Start' to start the service and 'Stop' to end it.
// The signingkey is usually the application account key
//
//	caCert is the CA certificate used to validate certs
//	signingKP used for signing JWT tokens by the server. usually the account key.
func NewNatsTokenizer(signingKP nkeys.KeyPair) *NatsAuthnTokenizer {

	tokenizer := &NatsAuthnTokenizer{signingKP: signingKP}
	return tokenizer
}
