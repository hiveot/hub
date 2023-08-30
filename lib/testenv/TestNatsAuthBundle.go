// Package testenv with managing certificates for testing
package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

const ServerAddress = "127.0.0.1"
const TestServiceID = "service1"
const TestDeviceID = "device1"
const TestUserID = "user1"

// TestNatsAuthBundle contains Nats test authentication keys and tokens
type TestNatsAuthBundle struct {
	// operator and account keys
	//OperatorNKey      nkeys.KeyPair
	OperatorJWT      string
	SystemAccountKey nkeys.KeyPair
	SystemAccountJWT string
	//SystemSigningNKey nkeys.KeyPair
	SystemUserNKey  nkeys.KeyPair
	SystemUserJWT   string
	SystemUserCreds []byte
	//
	AppAccountName string
	AppAccountKey  nkeys.KeyPair
	//AppSigningNKey nkeys.KeyPair
	AppAccountJWT string

	// test service
	ServiceID     string
	ServiceKey    nkeys.KeyPair // application services
	ServiceKeyPub string
	ServiceJWT    string
	ServiceCreds  []byte

	// Devices and services
	DeviceID     string
	DeviceKey    nkeys.KeyPair // IoT device key
	DeviceKeyPub string
	DeviceJWT    string
	DeviceCreds  []byte

	UserID     string
	UserKey    nkeys.KeyPair // end user key
	UserKeyPub string
	UserJWT    string
	UserCreds  []byte
}

// CreateTestAuthBundle creates a bundle of ca, server certificates and user keys for testing.
// The server cert is valid for the 127.0.0.1 ServerAddress only.
func CreateTestNatsAuthBundle() TestNatsAuthBundle {
	authBundle := TestNatsAuthBundle{}
	// life starts with the operator
	// the operator is signing the account keys
	// https://gist.github.com/renevo/8fa7282d441b46752c9151644a2e911a
	operatorNKey, _ := nkeys.CreateOperator()
	operatorPub, _ := operatorNKey.PublicKey()
	operatorClaims := jwt.NewOperatorClaims(operatorPub)
	operatorClaims.Name = "hiveotop"
	//// use a separate operator signing key that can be revoked without compromising the operator keys
	//operatorSigningNKey, _ := nkeys.CreateOperator()
	//operatorSigningPub, _ := operatorSigningNKey.PublicKey()
	//operatorClaims.SigningKeys.Add(operatorSigningPub)
	//authBundle.OperatorNKey = operatorNKey
	authBundle.OperatorJWT, _ = operatorClaims.Encode(operatorNKey)

	// the system account is used for monitoring
	systemAccountNKey, _ := nkeys.CreateAccount()
	systemAccountPub, _ := systemAccountNKey.PublicKey()
	systemSigningNKey, _ := nkeys.CreateAccount()
	systemSigningPub, _ := systemSigningNKey.PublicKey()
	systemAccountClaims := jwt.NewAccountClaims(systemAccountPub)
	systemAccountClaims.Name = "SYS"
	systemAccountClaims.SigningKeys.Add(systemSigningPub)
	systemAccountClaims.Exports = jwt.Exports{
		&jwt.Export{
			Name:                 "account-monitoring-services",
			Subject:              "$SYS.REQ.ACCOUNT.*.*",
			Type:                 jwt.Service,
			ResponseType:         jwt.ResponseTypeStream,
			AccountTokenPosition: 4,
			Info: jwt.Info{
				Description: "Custom account made by conservator",
				InfoURL:     "https://github.com/renevo/conservator",
			},
		},
		&jwt.Export{
			Name:                 "account-monitoring-streams",
			Subject:              "$SYS.ACCOUNT.*.>",
			Type:                 jwt.Stream,
			AccountTokenPosition: 3,
			Info: jwt.Info{
				Description: "Custom account made by conservator",
				InfoURL:     "https://github.com/renevo/conservator",
			},
		},
	}
	systemAccountJWT, _ := systemAccountClaims.Encode(operatorNKey)
	//authBundle.SystemSigningNKey = systemSigningNKey
	authBundle.SystemAccountKey = systemAccountNKey
	authBundle.SystemAccountJWT = systemAccountJWT

	// A user for the system account
	systemUserNKey, _ := nkeys.CreateUser()
	systemUserPub, _ := systemUserNKey.PublicKey()
	systemUserPriv, _ := systemUserNKey.Seed()
	systemUserClaims := jwt.NewUserClaims(systemUserPub)
	systemUserClaims.Name = "sys"
	systemUserClaims.IssuerAccount = systemAccountPub
	systemUserJWT, _ := systemUserClaims.Encode(systemSigningNKey)
	systemUserCreds, _ := jwt.FormatUserConfig(systemUserJWT, systemUserPriv)
	authBundle.SystemUserNKey = systemUserNKey
	authBundle.SystemUserJWT = systemUserJWT
	authBundle.SystemUserCreds = systemUserCreds

	// system account
	//operatorClaims.SystemAccount = systemAccountPub

	// the application uses a separate account key
	appAccountName := "AppAccount"
	appAccountKey, _ := nkeys.CreateAccount()
	appAccountPub, _ := appAccountKey.PublicKey()
	//appSigningNKey, _ := nkeys.CreateAccount()
	//appSigningPub, _ := appSigningNKey.PublicKey()
	appAccountClaims := jwt.NewAccountClaims(appAccountPub)
	appAccountClaims.Name = appAccountName
	//appAccountClaims.SigningKeys.Add(appSigningPub)
	// Enabling JetStream requires setting storage limits
	appAccountClaims.Limits.JetStreamLimits.DiskStorage = -1
	appAccountClaims.Limits.JetStreamLimits.MemoryStorage = -1
	appAccountJWT, _ := appAccountClaims.Encode(operatorNKey)
	authBundle.AppAccountName = appAccountName
	authBundle.AppAccountKey = appAccountKey
	//authBundle.AppSigningNKey = appSigningNKey
	authBundle.AppAccountJWT = appAccountJWT

	// test service keys created by the server account
	serviceNKey, _ := nkeys.CreateUser()
	servicePub := []string{">"}
	serviceSub := []string{">"}
	serviceJWT, serviceCreds := CreateUserCreds(
		TestServiceID, serviceNKey, appAccountKey, servicePub, serviceSub)
	authBundle.ServiceID = TestServiceID
	authBundle.ServiceKey = serviceNKey
	authBundle.ServiceKeyPub, _ = serviceNKey.PublicKey()
	authBundle.ServiceJWT = serviceJWT
	authBundle.ServiceCreds = serviceCreds

	// test device keys created by the server (account)
	deviceNKey, _ := nkeys.CreateUser()
	devicePub := []string{"things." + TestDeviceID + ".*.event.>"}
	deviceSub := []string{"_INBOX.>", "things." + TestDeviceID + ".*.action.>"}
	deviceJWT, deviceCreds := CreateUserCreds(
		TestDeviceID, deviceNKey, appAccountKey, devicePub, deviceSub)
	authBundle.DeviceID = TestDeviceID
	authBundle.DeviceKey = deviceNKey
	authBundle.DeviceKeyPub, _ = deviceNKey.PublicKey()
	authBundle.DeviceJWT = deviceJWT
	authBundle.DeviceCreds = deviceCreds

	// regular end-users can publish and subscribe to inbox and things
	userNKey, _ := nkeys.CreateUser()
	userPub := []string{"things.*.*.action.>"}
	userSub := []string{"_INBOX.>", "things.>"}
	userJWT, userCreds := CreateUserCreds(
		TestUserID, userNKey, appAccountKey, userPub, userSub)
	authBundle.UserID = TestUserID
	authBundle.UserKey = userNKey
	authBundle.UserKeyPub, _ = userNKey.PublicKey()
	authBundle.UserJWT = userJWT
	authBundle.UserCreds = userCreds

	return authBundle
}

// CreateUserCreds create a signed user JWT token and private credentials with pub/sub permissions
//
//	clientID is the client's authentication ID
//	userKey is the client's key pair
//	acctKey is the nkey of the signer, eg the account key
//	pub is the list of subjects allowed to publish or nil if not set here
//	sub is the list of subjects allowed to subscribe or nil if not set here
//
// This returns the public signed jwt token containing user claims, and the full credentials to be kept secret
func CreateUserCreds(clientID string,
	userKey nkeys.KeyPair,
	acctKey nkeys.KeyPair,
	pub []string, sub []string) (
	jwtToken string, creds []byte) {

	// device keys created by the server (account)
	pubKey, _ := userKey.PublicKey()
	privKey, _ := userKey.Seed()
	claims := jwt.NewUserClaims(pubKey)
	//claims.Subject = pubKey
	//-- in server mode this might work differently from operator mode
	// should appAccountName or public key be used???
	//claims.Audience = appAccountName
	claims.IssuerAccount, _ = acctKey.PublicKey()
	claims.Name = clientID
	claims.Tags.Add("clientType", auth.ClientTypeUser)
	// add identification and authorization to user
	// see also: https://natsbyexample.com/examples/auth/nkeys-jwts/go
	if pub != nil {
		claims.Permissions.Pub.Allow.Add(pub...)
	}
	if sub != nil {
		claims.Permissions.Sub.Allow.Add(sub...)
	}
	jwtToken, err := claims.Encode(acctKey)
	if err != nil {
		panic("cant create jwt key:" + err.Error())
	}
	creds, _ = jwt.FormatUserConfig(jwtToken, privKey)
	return jwtToken, creds
}
