package service

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/core/authn"
	"github.com/hiveot/hub/core/authn/service/unpwstore"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
	"golang.org/x/exp/rand"
	"golang.org/x/exp/slog"
	"strings"
	"time"
)

// AuthnService provides the capabilities to manage and use authentication services
// This implements the IAuthnService interface
// TODO: should this use action messages directly to allow additional validation of the caller???
type AuthnService struct {
	//accountName string

	signingKP nkeys.KeyPair
	// the signingJWT claims used for signing JWT user tokens
	// This must be a key known to the server for validation
	//signingJWT string
	// password storage
	pwStore unpwstore.IUnpwStore
	// ca certificate for cert validation
	caCert *x509.Certificate
}

// AddDevice adds a device
//func (svc *AuthnService) AddDevice(clientID string, name string) (token string, err error) {
//
//	exists := svc.pwStore.Exists(clientID)
//	if exists {
//		return "", fmt.Errorf("device with clientID '%s' already exists", clientID)
//	}
//	err = svc.pwStore.SetName(clientID, name)
//	return err
//}

// AddService adds a svc
//func (svc *AuthnService) AddService(clientID string, name string) (token string, err error) {
//
//	exists := svc.pwStore.Exists(clientID)
//	if exists {
//		return "", fmt.Errorf("svc with clientID '%s' already exists", clientID)
//	}
//	err = svc.pwStore.SetName(clientID, name)
//	return err
//}

// AddUser adds a new user for password authentication
func (svc *AuthnService) AddUser(userID string, userName string, password string) (err error) {

	exists := svc.pwStore.Exists(userID)
	if exists {
		return fmt.Errorf("user with clientID '%s' already exists", userID)
	}
	err = svc.pwStore.SetPassword(userID, password)
	_ = svc.pwStore.SetName(userID, userName)
	return err
}

// CreateUserToken create a new jwt token for connecting to the server
func (svc *AuthnService) CreateUserToken(userID string, userName string, pubKey string, validitySec uint) (string, error) {
	if validitySec == 0 {
		validitySec = authn.DefaultUserTokenValiditySec
	}

	// create jwt claims that identifies the user and its permissions
	userClaims := jwt.NewUserClaims(pubKey)
	// can't use claim ID as it is replaced by a hash by Encode(kp)
	userClaims.Name = userID
	userClaims.Type = authn.ClientTypeUser
	userClaims.User.Tags = append(userClaims.User.Tags, "userName:"+userName)
	userClaims.IssuedAt = time.Now().Unix()
	userClaims.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()

	// default size
	userClaims.Limits.Data = 1024 * 1024 * 1024 // max data this client can ... do?
	// users can publish actions
	userClaims.Permissions.Pub.Allow.Add("groups.*.*.action.>")
	// users can subscribe to group events
	userClaims.Permissions.Sub.Allow.Add("groups.*.*.event.>")
	// users can receive replies in their inbox
	userClaims.Permissions.Sub.Allow.Add("_INBOX.>")
	// the subject MUST be the public key
	userClaims.Subject = pubKey

	// sign the claims with the client's private key
	userJWT, err := userClaims.Encode(svc.signingKP)

	// create a decorated jwt/nkey pair for future use.
	// the caller must first be authenticated before giving it a jwt token.
	//creds, _ := jwt.FormatUserConfig(userJWT, userSeed)
	//return string(creds), err
	return userJWT, err
}

// CreateDeviceToken create a new jwt token for connecting IoT devices to the server
func (svc *AuthnService) CreateDeviceToken(deviceID string, pubKey string, validitySec uint) (string, error) {
	if validitySec == 0 {
		validitySec = authn.DefaultDeviceTokenValiditySec
	}
	// create jwt claims that identifies the user and its permissions
	userClaims := jwt.NewUserClaims(pubKey)
	// can't use claim ID as it is replaced by a hash by Encode(kp)
	userClaims.Name = deviceID
	userClaims.Type = authn.ClientTypeDevice
	userClaims.IssuedAt = time.Now().Unix()
	userClaims.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()

	// default size
	userClaims.Limits.Data = 1 * 1024 * 1024 // max data this client can ... do what?
	// devices can publish events of which they are the publisher
	userClaims.Permissions.Pub.Allow.Add("things." + deviceID + ".*.event.>")
	// devices can subscribe to actions aimed at them
	userClaims.Permissions.Sub.Allow.Add("things." + deviceID + ".*.action.>")
	// devices can publish replies to user inbox  - TBD: is this needed?
	userClaims.Permissions.Pub.Allow.Add("_INBOX.>")
	// the claims subject MUST be the device public key
	userClaims.Subject = pubKey

	// sign the claims with the service signing key
	userJWT, err := userClaims.Encode(svc.signingKP)

	// create a decorated jwt/nkey pair for future use.
	// the caller must first be authenticated before giving it a jwt token.
	//creds, _ := jwt.FormatUserConfig(userJWT, userSeed)
	//return string(creds), err
	return userJWT, err
}

// CreateServiceToken create a new jwt token for connecting services to the server
func (svc *AuthnService) CreateServiceToken(serviceID string, pubKey string, validitySec uint) (string, error) {
	if validitySec == 0 {
		validitySec = authn.DefaultServiceTokenValiditySec
	}

	// create jwt claims that identifies the service and its permissions
	userClaims := jwt.NewUserClaims(pubKey)
	// can't use claim ID as it is replaced by a hash by Encode(kp)
	userClaims.Name = serviceID
	userClaims.Type = authn.ClientTypeService
	userClaims.IssuedAt = time.Now().Unix()
	userClaims.Expires = time.Now().Add(time.Duration(validitySec) * time.Second).Unix()

	// default size
	userClaims.Limits.Data = 1 * 1024 * 1024 // max data this client can ... do what?
	// services can publish events of which they are the publisher
	userClaims.Permissions.Pub.Allow.Add("things." + serviceID + ".*.event.>")
	// services can subscribe to any events
	userClaims.Permissions.Sub.Allow.Add("things.*.*.event.>")
	// services can subscribe to actions aimed at them
	userClaims.Permissions.Sub.Allow.Add("things." + serviceID + ".*.action.>")
	// devices can publish replies to user inbox  - TBD: is this needed?
	userClaims.Permissions.Pub.Allow.Add("_INBOX.>")
	// the claims subject MUST be the device public key
	userClaims.Subject = pubKey

	// sign the claims with the service signing key
	userJWT, err := userClaims.Encode(svc.signingKP)

	// create a decorated jwt/nkey pair for future use.
	// the caller must first be authenticated before giving it a jwt token.
	//creds, _ := jwt.FormatUserConfig(userJWT, userSeed)
	//return string(creds), err
	return userJWT, err
}

// GetProfile returns the current connected user's profile
// technically the same as GetClientProfile, except that the latter can provide
// different info for managers. Not making assum
func (svc *AuthnService) GetProfile(clientID string) (profile authn.ClientProfile, err error) {
	//upa.profileStore[profile.LoginID] = profile
	entry, err := svc.pwStore.GetEntry(clientID)
	if err != nil {
		return profile, fmt.Errorf("can't get profile from %s: %w", clientID, err)
	}
	updatedStr := time.Unix(entry.Updated, 0).Format(vocab.ISO8601Format)
	profile.ClientID = entry.LoginID
	profile.Name = entry.UserName
	profile.Updated = updatedStr
	return profile, err

}

// GetClientProfile returns a client's profile
func (svc *AuthnService) GetClientProfile(clientID string) (profile authn.ClientProfile, err error) {
	//upa.profileStore[profile.LoginID] = profile
	entry, err := svc.pwStore.GetEntry(clientID)
	if err != nil {
		return profile, fmt.Errorf("can't get profile from %s: %w", clientID, err)
	}
	updatedStr := time.Unix(entry.Updated, 0).Format(vocab.ISO8601Format)
	profile.ClientID = entry.LoginID
	profile.Name = entry.UserName
	profile.Updated = updatedStr
	return profile, err

}

// GeneratePassword with upper, lower, numbers and special characters
func (svc *AuthnService) GeneratePassword(length int, useSpecial bool) (password string) {
	const charsLow = "abcdefghijklmnopqrstuvwxyz"
	const charsUpper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	const charsSpecial = "!#$%&*+-./:=?@^_"
	const numbers = "0123456789"
	var pool = []rune(charsLow + numbers + charsUpper)

	if length < 2 {
		length = 8
	}
	if useSpecial {
		pool = append(pool, []rune(charsSpecial)...)
	}
	rand.Seed(uint64(time.Now().Unix()))
	//pwchars := make([]string, length)
	pwchars := strings.Builder{}

	for i := 0; i < length; i++ {
		pos := rand.Intn(len(pool))
		pwchars.WriteRune(pool[pos])
	}
	password = pwchars.String()
	return password
}

// ListClients provide a list of users and their info
func (svc *AuthnService) ListClients() (profiles []authn.ClientProfile, err error) {
	pwEntries, err := svc.pwStore.List()
	if err != nil {
		return nil, fmt.Errorf("error listing clients: %w", err)
	}
	profiles = make([]authn.ClientProfile, len(pwEntries))
	for i, entry := range pwEntries {
		updatedStr := time.Unix(entry.Updated, 0).Format(vocab.ISO8601Format)
		profile := authn.ClientProfile{
			ClientID: entry.LoginID,
			Name:     entry.UserName,
			Updated:  updatedStr,
		}
		profiles[i] = profile
	}
	slog.Info("ListClients", "nr clients", len(profiles))
	return profiles, err
}

// NewToken creates a new jwt auth token based on userName, password and public key
// This returns a short-lived auth token that can be used to connect to the message server
// The token can be refreshed to extend it without requiring a login password.
func (svc *AuthnService) NewToken(clientID string, password string, pubKey string) (jwtToken string, err error) {
	slog.Info("creating new token", slog.String("clientID", clientID))
	entry, err := svc.pwStore.VerifyPassword(clientID, password)
	if err != nil {
		return "", fmt.Errorf("error getting new token for %s: %w", clientID, err)
	}
	// when valid, provide the tokens
	jwtToken, err = svc.CreateUserToken(
		clientID, entry.UserName, pubKey, authn.DefaultUserTokenValiditySec)
	return jwtToken, err
}

// Refresh an authentication token
// This returns a refreshed token that can be used to connect to the messaging server
// the old token must be a valid jwt token belonging to the clientID
func (svc *AuthnService) Refresh(clientID string, oldToken string) (newToken string, err error) {
	slog.Info("refresh token", "clientID", clientID)
	// verify the token
	entry, claims, err := svc.ValidateToken(clientID, oldToken)
	if err != nil {
		return "", fmt.Errorf("error validating oldToken of client %s: %w", clientID, err)
	}
	pubKey := claims.Claims().Subject
	newToken, err = svc.CreateUserToken(clientID, entry.UserName, pubKey, authn.DefaultUserTokenValiditySec)
	return newToken, err
}

// RemoveClient removes a user and disables login
// Existing tokens are immediately expired (tbd)
func (svc *AuthnService) RemoveClient(clientID string) (err error) {
	slog.Info("removing client", "clientID", clientID)
	err = svc.pwStore.Remove(clientID)
	if err != nil {
		return fmt.Errorf("error removing client %s: %w", clientID, err)
	}
	return err
}

// ResetPassword sets the client password
func (svc *AuthnService) ResetPassword(clientID, newPassword string) error {
	return svc.pwStore.SetPassword(clientID, newPassword)
}

// Start the svc, open the password store
func (svc *AuthnService) Start() error {
	slog.Info("starting authn svc")

	//authKey, err := svc.config.GetAuthKey()
	//if err != nil {
	//	return err
	//}

	err := svc.pwStore.Open()
	if err != nil {
		return fmt.Errorf("error starting authn svc: %w", err)
	}
	//err = svc.hc.ConnectWithJWT(svc.config.ServerURL, []byte(authKey), svc.caCert)
	return err
}

func (svc *AuthnService) Stop() error {
	slog.Info("stopping AuthnService")
	svc.pwStore.Close()
	return nil
}

// UpdateName updates a user's name
func (svc *AuthnService) UpdateName(clientID string, name string) (err error) {
	slog.Info("update client name", slog.String("clientID", clientID))
	exists := svc.pwStore.Exists(clientID)
	if !exists {
		return fmt.Errorf("user with loginID '%s' does not exist", clientID)
	}
	err = svc.pwStore.SetName(clientID, name)
	return err
}

// UpdatePassword changes the client password
func (svc *AuthnService) UpdatePassword(clientID, newPassword string) error {
	slog.Info("update password", slog.String("clientID", clientID))
	return svc.pwStore.SetPassword(clientID, newPassword)
}

// ValidateCert verifies that the given certificate belongs to the client
// and is signed by our CA.
// - CN is clientID (todo: other means?)
// - Cert validates against the svc CA
// This is intended for a local setup that use a self-signed CA.
// The use of JWT keys is recommended over certs as this isn't a domain name validation problem.
func (svc *AuthnService) ValidateCert(clientID string, clientCertPEM string) error {
	certBlock, _ := pem.Decode([]byte(clientCertPEM))
	if certBlock == nil {
		return fmt.Errorf("invalid cert pem for client '%s. decode failed", clientID)
	}
	clientCert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return err
	}
	// verify the cert against the CA
	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(svc.caCert)
	verifyOpts := x509.VerifyOptions{
		Roots:     caCertPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	_, err = clientCert.Verify(verifyOpts)

	// verify the certs belongs to the clientID
	certUser := clientCert.Subject.CommonName
	if certUser != clientID {
		return fmt.Errorf("cert user '%s' doesnt match client '%s'", certUser, clientID)
	}
	return nil
}

// ValidateNatsJWT checks if the given token belongs the the user ID and is valid
//   - verify if jwtToken is a valid token
//   - validate the token isn't expired
//   - verify the user's public key's nonce based signature
//     this can only be signed when the user has its private key
//   - verify the issuer is the signing/account key.
func (svc *AuthnService) ValidateNatsJWT(
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

	// do we know this user?
	entry, err := svc.pwStore.GetEntry(clientID)
	if err != nil {
		return fmt.Errorf("unknown user %s", clientID)
	}
	// todo, store user's public key
	_ = entry
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

// ValidatePassword verifies the given username password is valid
func (svc *AuthnService) ValidatePassword(clientID string, password string) error {
	_, err := svc.pwStore.VerifyPassword(clientID, password)
	if err != nil {
		return err
	}
	return nil
}

// ValidateToken validates whether a valid jwt token was given
// This validates:
// - if the token is a JWT token
// - if the token's ID matches the clientID
// - if the claims issuer is the public signing key
// - if the token is signed by the signing key on record
// - if the token contains the client public key (subject)
// - if the token claim type is a user claim
// - if the token isn't expired
// - if the user exists
// TODO: verify issuer
func (svc *AuthnService) ValidateToken(clientID string, jwtToken string) (
	entry unpwstore.PasswordEntry, claims jwt.Claims, err error) {
	//slog.Info("validate token", slog.String("clientID",clientID))

	entry, err = svc.pwStore.GetEntry(clientID)
	_ = entry
	if err != nil {
		return entry, nil, fmt.Errorf("unknown user %s", clientID)
	}
	claims, err = jwt.Decode(jwtToken)
	if err != nil {
		return entry, nil, fmt.Errorf("invalid token of client %s: %w", clientID, err)
	}
	cd := claims.Claims()
	//claims, err = jwt.DecodeGeneric(jwtToken)
	// issuer must be known
	signingPub, _ := svc.signingKP.PublicKey()
	if cd.Issuer != signingPub {
		return entry, claims, errors.New("unknown issuer")
	}
	if claims.ClaimType() != jwt.UserClaim {
		return entry, claims, errors.New("Token is not a user token of client " + clientID)
	}
	if cd.Name != clientID {
		slog.Warn("Token from different user",
			"token ID", cd.ID, "token name", cd.Name, "clientID", clientID)
		return entry, nil, errors.New("Token is from a different client, not" + clientID)
	}
	// TODO: check if the token public key matches that of the claimed user
	//if cd.Subject != entry.PubKey {
	//}
	//FIXME: validate that the jwt token is properly signed
	//if !validateSignature() {
	//}
	vr := jwt.ValidationResults{}
	cd.Validate(&vr)

	if !vr.IsEmpty() {
		err = errors.New("Invalid token: " + vr.Errors()[0].Error())
		return entry, nil, err
	}
	return entry, claims, nil
}

// NewAuthnService creates new instance of the svc
// Call 'Start' to start the service and 'Stop' to end it.
// The signingkey is usually the application account key
//
//	pwStore is the store for users and encrypted passwords
//	caCert is the CA certificate used to validate certs
//	signingKP used for signing JWT tokens by the server. usually the account key.
func NewAuthnService(
	pwStore unpwstore.IUnpwStore,
	caCert *x509.Certificate,
	signingKP nkeys.KeyPair,
) *AuthnService {

	svc := &AuthnService{
		//accountName,
		caCert:    caCert,
		pwStore:   pwStore,
		signingKP: signingKP,
	}
	return svc
}
