package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/nkeys"
	"golang.org/x/crypto/bcrypt"
)

// ID
var TestDevice1ID = "device1"
var TestDevice1NKey, _ = nkeys.CreateUser()
var TestDevice1NPub, _ = TestDevice1NKey.PublicKey()
var TestDevice1Key, TestDevice1Pub = certs.CreateECDSAKeys()

var TestThing1ID = "thing1"
var TestThing2ID = "thing2"

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)

var TestAdminUserID = "admin"
var TestAdminUserNKey, _ = nkeys.CreateUser()
var TestAdminUserNPub, _ = TestAdminUserNKey.PublicKey()
var TestAdminUserKey, TestAdminUserPub = certs.CreateECDSAKeys()

var TestService1ID = "service1"
var TestService1NKey, _ = nkeys.CreateUser()
var TestService1NPub, _ = TestService1NKey.PublicKey()
var TestService1Key, TestService1Pub = certs.CreateECDSAKeys()

// TestClients contains test users and devices
func CreateTestClients(core string) []msgserver.ClientAuthInfo {
	if core == "nats" {
		return []msgserver.ClientAuthInfo{
			{
				ClientID:   TestAdminUserID,
				ClientType: auth.ClientTypeUser,
				PubKey:     TestAdminUserNPub,
				Role:       auth.ClientRoleAdmin,
			},
			{
				ClientID:   TestDevice1ID,
				ClientType: auth.ClientTypeDevice,
				PubKey:     TestDevice1NPub,
				Role:       auth.ClientRoleDevice,
			},
			{
				ClientID:     TestUser1ID,
				ClientType:   auth.ClientTypeUser,
				PasswordHash: string(TestUser1bcrypt),
				Role:         auth.ClientRoleViewer,
			},
			{
				ClientID:   TestService1ID,
				ClientType: auth.ClientTypeService,
				PubKey:     TestService1NPub,
				Role:       auth.ClientRoleAdmin,
			},
		}
	}
	//mqtt keys
	return []msgserver.ClientAuthInfo{
		{
			ClientID:   TestAdminUserID,
			ClientType: auth.ClientTypeUser,
			PubKey:     TestAdminUserPub,
			Role:       auth.ClientRoleAdmin,
		},
		{
			ClientID:   TestDevice1ID,
			ClientType: auth.ClientTypeDevice,
			PubKey:     TestDevice1Pub,
			Role:       auth.ClientRoleDevice,
		},
		{
			ClientID:     TestUser1ID,
			ClientType:   auth.ClientTypeUser,
			PasswordHash: string(TestUser1bcrypt),
			Role:         auth.ClientRoleViewer,
		},
		{
			ClientID:   TestService1ID,
			ClientType: auth.ClientTypeService,
			PubKey:     TestService1Pub,
			Role:       auth.ClientRoleAdmin,
		},
	}

}
