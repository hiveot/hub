// Package testenv with managing certificates for testing
package testenv

import (
	"crypto/ecdsa"
	"crypto/tls"
	"crypto/x509"
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/jwt/v2"
	"github.com/nats-io/nkeys"
)

const ServerAddress = "127.0.0.1"
const TestServiceID = "service1"
const TestDeviceID = "device1"
const TestUserID = "user1"

// TestAuthBundle contain test certificates for CA and server
type TestAuthBundle struct {
	CaCert *x509.Certificate
	CaKey  *ecdsa.PrivateKey

	// Nats server certificate
	ServerKey  *ecdsa.PrivateKey
	ServerCert *tls.Certificate
	// operator and account keys
	//OperatorNKey      nkeys.KeyPair
	OperatorJWT       string
	SystemAccountNKey nkeys.KeyPair
	SystemAccountJWT  string
	//SystemSigningNKey nkeys.KeyPair
	SystemUserNKey  nkeys.KeyPair
	SystemUserJWT   string
	SystemUserCreds []byte
	//
	AppAccountName string
	AppAccountNKey nkeys.KeyPair
	//AppSigningNKey nkeys.KeyPair
	AppAccountJWT string

	// test service
	ServiceID    string
	ServiceNKey  nkeys.KeyPair // application services
	ServiceJWT   string
	ServiceCreds []byte

	// Devices and services
	DeviceID    string
	DeviceNKey  nkeys.KeyPair // IoT device key
	DeviceJWT   string
	DeviceCreds []byte

	UserID    string
	UserNKey  nkeys.KeyPair // end user key
	UserJWT   string
	UserCreds []byte
}

// CreateTestAuthBundle creates a bundle of ca, server certificates and user keys for testing.
// The server cert is valid for the 127.0.0.1 ServerAddress only.
func CreateTestAuthBundle() TestAuthBundle {
	authBundle := TestAuthBundle{}
	// Setup CA and server TLS certificates
	authBundle.CaCert, authBundle.CaKey, _ = certs.CreateCA("testing", 1)
	authBundle.ServerKey = certs.CreateECDSAKeys()

	names := []string{ServerAddress}
	serverCert, err := certs.CreateServerCert(
		TestServiceID, "server",
		&authBundle.ServerKey.PublicKey,
		names, 1,
		authBundle.CaCert, authBundle.CaKey)
	if err != nil {
		panic("unable to create server cert: " + err.Error())
	}
	authBundle.ServerCert = certs.X509CertToTLS(serverCert, authBundle.ServerKey)

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
	authBundle.SystemAccountNKey = systemAccountNKey
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
	appAccountNKey, _ := nkeys.CreateAccount()
	appAccountPub, _ := appAccountNKey.PublicKey()
	//appSigningNKey, _ := nkeys.CreateAccount()
	//appSigningPub, _ := appSigningNKey.PublicKey()
	appAccountClaims := jwt.NewAccountClaims(appAccountPub)
	appAccountClaims.Name = appAccountName
	//appAccountClaims.SigningKeys.Add(appSigningPub)
	// Enabling JetStream requires setting storage limits
	appAccountClaims.Limits.JetStreamLimits.DiskStorage = -1
	appAccountClaims.Limits.JetStreamLimits.MemoryStorage = -1
	//appAccountClaims.Subject = appAccountPub
	//appAccountClaims.Exports = jwt.Exports{
	//	&jwt.Export{
	//		Name:                 "account-monitoring-services",
	//		Subject:              "$SYS.REQ.ACCOUNT.*.*",
	//		Type:                 jwt.Service,
	//		ResponseType:         jwt.ResponseTypeStream,
	//		AccountTokenPosition: 4,
	//		Info: jwt.Info{
	//			Description: "Custom account made by conservator",
	//			InfoURL:     "https://github.com/renevo/conservator",
	//		},
	//	},
	//	&jwt.Export{
	//		Name:                 "account-monitoring-streams",
	//		Subject:              "$SYS.ACCOUNT.*.>",
	//		Type:                 jwt.Stream,
	//		AccountTokenPosition: 3,
	//		Info: jwt.Info{
	//			Description: "Custom account made by conservator",
	//			InfoURL:     "https://github.com/renevo/conservator",
	//		},
	//	},
	//}
	appAccountJWT, _ := appAccountClaims.Encode(operatorNKey)
	authBundle.AppAccountName = appAccountName
	authBundle.AppAccountNKey = appAccountNKey
	//authBundle.AppSigningNKey = appSigningNKey
	authBundle.AppAccountJWT = appAccountJWT

	// test service keys created by the server account
	serviceNKey, _ := nkeys.CreateUser()
	servicePub := []string{">"}
	serviceSub := []string{">"}
	serviceJWT, serviceCreds := CreateUserCreds(
		TestServiceID, serviceNKey, appAccountNKey, appAccountPub, appAccountName, servicePub, serviceSub)
	authBundle.ServiceID = TestServiceID
	authBundle.ServiceNKey = serviceNKey
	authBundle.ServiceJWT = serviceJWT
	authBundle.ServiceCreds = serviceCreds

	// test device keys created by the server (account)
	deviceNKey, _ := nkeys.CreateUser()
	devicePub := []string{"things." + TestDeviceID + ".*.event.>"}
	deviceSub := []string{"_INBOX.>", "things." + TestDeviceID + ".*.action.>"}
	deviceJWT, deviceCreds := CreateUserCreds(
		TestDeviceID, deviceNKey, appAccountNKey, appAccountPub, appAccountName, devicePub, deviceSub)
	authBundle.DeviceID = TestDeviceID
	authBundle.DeviceNKey = deviceNKey
	authBundle.DeviceJWT = deviceJWT
	authBundle.DeviceCreds = deviceCreds

	// regular end-users can publish and subscribe to inbox and things
	userNKey, _ := nkeys.CreateUser()
	userPub := []string{"things.*.*.action.>"}
	userSub := []string{"_INBOX.>", "things.>"}
	userJWT, userCreds := CreateUserCreds(
		TestUserID, userNKey, appAccountNKey, appAccountPub, appAccountName, userPub, userSub)
	authBundle.UserID = TestUserID
	authBundle.UserNKey = userNKey
	authBundle.UserJWT = userJWT
	authBundle.UserCreds = userCreds

	return authBundle
}

// CreateUserCreds create a signed user JWT token and private credentials with pub/sub permissions
//
//	id is the client's authentication ID
//	nkey is the client's key pair
//	signer is the nkey of the signer, eg the account key
//	pub is the list of subjects allowed to publish or nil if not set here
//	sub is the list of subjects allowed to subscribe or nil if not set here
//
// This returns the public signed jwt token containing user claims, and the full credentials to be kept secret
func CreateUserCreds(id string, keys nkeys.KeyPair,
	signer nkeys.KeyPair, appAccountPub string, appAccountName string,
	pub []string, sub []string) (
	jwtToken string, creds []byte) {

	// device keys created by the server (account)
	pubKey, _ := keys.PublicKey()
	privKey, _ := keys.Seed()
	claims := jwt.NewUserClaims(pubKey)
	claims.Subject = pubKey
	//-- in server mode this might work differently from operator mode
	// should appAccountName or public key be used???
	//claims.Audience = appAccountName
	claims.IssuerAccount = appAccountPub
	//
	claims.Name = id
	// add identification and authorization to user
	// see also: https://natsbyexample.com/examples/auth/nkeys-jwts/go
	if pub != nil {
		claims.Pub.Allow.Add(pub...)
	}
	if sub != nil {
		claims.Sub.Allow.Add(sub...)
	}
	jwtToken, err := claims.Encode(signer)
	if err != nil {
		panic("cant create jwt key:" + err.Error())
	}
	creds, _ = jwt.FormatUserConfig(jwtToken, privKey)
	return jwtToken, creds
}
