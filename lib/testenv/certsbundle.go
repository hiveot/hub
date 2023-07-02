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

const TestServerID = "server1"
const ServerAddress = "127.0.0.1"
const TestDeviceID = "device1"
const TestUserID = "user1"

// TestCerts contain test certificates for CA and server
type TestCerts struct {
	CaCert *x509.Certificate
	CaKey  *ecdsa.PrivateKey

	// Operator
	OperatorNKey nkeys.KeyPair
	AccountNKey  nkeys.KeyPair

	// Nats server certificate and keys
	ServerCert  *tls.Certificate
	ServerID    string
	ServerKey   *ecdsa.PrivateKey
	ServerNKey  nkeys.KeyPair // For use by nats server
	ServerJWT   string
	ServerCreds []byte

	// tbd operator or account key for server?
	// Service, device and user keys
	//ServiceNKey  nkeys.KeyPair // application services
	//ServiceJWT   string
	//ServiceCreds []byte

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

// CreateAuthBundle creates a bundle of ca, server certificates and client keys for testing.
// The server cert is valid for ServerAddress only.
func CreateAuthBundle() TestCerts {
	testCerts := TestCerts{
		ServerID: TestServerID,
		DeviceID: TestDeviceID,
		UserID:   TestUserID,
	}
	testCerts.CaCert, testCerts.CaKey, _ = certs.CreateCA("testing", 1)
	testCerts.ServerKey = certs.CreateECDSAKeys()

	names := []string{ServerAddress}
	serverCert, err := certs.CreateServerCert(
		testCerts.ServerID, "server",
		&testCerts.ServerKey.PublicKey,
		names, 1,
		testCerts.CaCert, testCerts.CaKey)
	if err == nil {
		testCerts.ServerCert = certs.X509CertToTLS(serverCert, testCerts.ServerKey)
	}

	// NATS authentication using nkeys and JWT claims
	testCerts.OperatorNKey, _ = nkeys.CreateOperator()
	testCerts.AccountNKey, _ = nkeys.CreateAccount()
	//operatorPub, _ := testCerts.OperatorNKey.PublicKey()
	//operatorSeed, _ := testCerts.OperatorNKey.Seed()

	// The server (account?) created by the operator
	testCerts.ServerNKey, _ = nkeys.CreateServer()
	serverPub, _ := testCerts.ServerNKey.PublicKey()
	serverSeed, _ := testCerts.ServerNKey.Seed()
	serverClaims := jwt.NewAccountClaims(serverPub)
	// FIXME: what?
	serverClaims.Subject, _ = testCerts.AccountNKey.PublicKey()
	serverClaims.Name = testCerts.ServerID
	serverClaims.Limits.JetStreamLimits.DiskStorage = 1024 * 1024 * 1024  // 1GB disk for testing
	serverClaims.Limits.JetStreamLimits.MemoryStorage = 1024 * 1024 * 100 // 100MB memory for testing
	testCerts.ServerJWT, err = serverClaims.Encode(testCerts.AccountNKey)
	if err != nil {
		panic(err.Error())
	}
	testCerts.ServerCreds, _ = jwt.FormatUserConfig(testCerts.ServerJWT, serverSeed)

	// Services keys created by the server (account)
	//testCerts.ServiceNKey, _ = nkeys.CreateUser()
	//servicePub, _ := testCerts.ServerNKey.PublicKey()
	//serviceSeed, _ := testCerts.ServerNKey.Seed()
	//serviceClaims := jwt.NewAccountClaims(servicePub)
	//serviceClaims.Name = "service1"
	//testCerts.ServiceJWT, _ = serviceClaims.Encode(testCerts.ServerNKey)
	//testCerts.ServiceCreds, _ = jwt.FormatUserConfig(testCerts.ServiceJWT, serviceSeed)

	// device keys created by the server (account)
	testCerts.DeviceNKey, _ = nkeys.CreateUser()
	devicePub, _ := testCerts.DeviceNKey.PublicKey()
	deviceSeed, _ := testCerts.DeviceNKey.Seed()
	deviceClaims := jwt.NewAccountClaims(devicePub)
	// FIXME: what?
	//deviceClaims.Subject = devicePub
	deviceClaims.Subject, _ = testCerts.AccountNKey.PublicKey()
	deviceClaims.Name = testCerts.DeviceID
	testCerts.DeviceJWT, err = deviceClaims.Encode(testCerts.AccountNKey)
	if err != nil {
		panic("cant create device jwt key:" + err.Error())
	}
	testCerts.DeviceCreds, _ = jwt.FormatUserConfig(testCerts.DeviceJWT, deviceSeed)

	// add identification and authorization to user
	// see also: https://natsbyexample.com/examples/auth/nkeys-jwts/go
	testCerts.UserNKey, _ = nkeys.CreateUser()
	userPub, _ := testCerts.UserNKey.PublicKey()
	userClaims := jwt.NewUserClaims(userPub)
	userClaims.Name = testCerts.UserID   // name vs ID?
	userClaims.Limits.Data = 1024 * 1024 // not sure how this is used
	userClaims.Permissions.Pub.Allow.Add(">")
	userClaims.Permissions.Sub.Allow.Add("_INBOX.>")
	// sign the JWT by the server (operator?) and produce the decorated credentials that can be written to a file
	// what does 'decorated' mean?
	testCerts.UserJWT, _ = userClaims.Encode(testCerts.ServerNKey)
	userSeed, _ := testCerts.UserNKey.Seed()
	testCerts.UserCreds, _ = jwt.FormatUserConfig(testCerts.UserJWT, userSeed)

	return testCerts
}
