package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/nats-io/nkeys"
	"golang.org/x/crypto/bcrypt"
)

var TestDevice1ID = "device1"
var TestDevice1Key, _ = nkeys.CreateUser()
var TestDevice1Pub, _ = TestDevice1Key.PublicKey()
var TestThing1ID = "thing1"
var TestThing2ID = "thing2"

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)

var TestAdminUserID = "admin"
var TestAdminUserKey, _ = nkeys.CreateUser()
var TestAdminUserPub, _ = TestAdminUserKey.PublicKey()

var TestService1ID = "service1"
var TestService1Key, _ = nkeys.CreateUser()
var TestService1Pub, _ = TestService1Key.PublicKey()

// TestClients contains test users and devices
var TestClients = []msgserver.ClientAuthInfo{
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
