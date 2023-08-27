package testenv

import (
	"github.com/hiveot/hub/api/go/authn"
	"github.com/hiveot/hub/api/go/authz"
	"github.com/hiveot/hub/core/msgserver/natsnkeyserver"
	"github.com/hiveot/hub/lib/certs"
	"github.com/nats-io/nkeys"
	"golang.org/x/crypto/bcrypt"
	"os"
	"path"
)

var TestDevice1ID = "device1"
var TestDevice1Key, _ = nkeys.CreateUser()
var TestDevice1Pub, _ = TestDevice1Key.PublicKey()
var TestThing1ID = "thing1"
var TestThing2ID = "thing2"

var TestUser1ID = "user1"
var TestUser1Pass = "pass1"
var TestUser1bcrypt, _ = bcrypt.GenerateFromPassword([]byte(TestUser1Pass), 0)

var TestUser2ID = "user2"
var TestUser2Key, _ = nkeys.CreateUser()
var TestUser2Pub, _ = TestUser2Key.PublicKey()

var TestService1ID = "service1"
var TestService1Key, _ = nkeys.CreateUser()
var TestService1Pub, _ = TestService1Key.PublicKey()

var TestGroup1ID = "group1"
var TestGroup2ID = "group2"

// test users
var TestClients = []authn.AuthnEntry{
	{ClientProfile: authn.ClientProfile{
		ClientID:    TestDevice1ID,
		ClientType:  authn.ClientTypeDevice,
		DisplayName: "device1 1",
		PubKey:      TestDevice1Pub,
	}},
	{ClientProfile: authn.ClientProfile{
		ClientID:    TestUser1ID,
		ClientType:  authn.ClientTypeUser,
		DisplayName: "user 1",
	}, PasswordHash: string(TestUser1bcrypt),
	},
	{ClientProfile: authn.ClientProfile{
		ClientID:    TestUser2ID,
		ClientType:  authn.ClientTypeUser,
		DisplayName: "user 2",
		PubKey:      TestUser2Pub,
	}},
	{ClientProfile: authn.ClientProfile{
		ClientID:    TestService1ID,
		ClientType:  authn.ClientTypeService,
		DisplayName: "service 1",
		PubKey:      TestService1Pub,
	}},
}
var TestRoles = map[string]authz.UserRoleMap{
	TestUser1ID:    {TestGroup1ID: authz.UserRoleViewer},
	TestUser2ID:    {TestGroup1ID: authz.UserRoleOperator},
	TestService1ID: {TestGroup1ID: authz.UserRoleManager, TestGroup2ID: authz.UserRoleViewer},
	//TestDevice1ID:  {TestGroup1ID: authz.GroupRoleIotDevice, TestGroup2ID: authz.GroupRoleIotDevice},
	//TestThing1ID:   {TestGroup1ID: authz.GroupRoleThing, TestGroup2ID: authz.GroupRoleThing},
}

// StartNatsTestServer generate a test configuration and starts a NKeys based nats test server
// A new temporary storage directory is used.
func StartNatsTestServer() (
	clientURL string,
	hubNatsServer *natsnkeyserver.NatsNKeyServer,
	certBundle certs.TestCertBundle,
	config *natsnkeyserver.NatsServerConfig, err error) {

	certBundle = certs.CreateTestCertBundle()

	serverCfg := &natsnkeyserver.NatsServerConfig{
		Port:   9990,
		CaCert: certBundle.CaCert,
		CaKey:  certBundle.CaKey,
		//ServerCert: certBundle.ServerCert, // auto generate
		//Debug: true,
	}
	//
	tmpDir := path.Join(os.TempDir(), "nats-testserver")
	_ = os.RemoveAll(tmpDir)
	err = serverCfg.Setup("", tmpDir, false)
	if err == nil {
		hubNatsServer = natsnkeyserver.NewNatsNKeyServer(serverCfg)
		clientURL, err = hubNatsServer.Start()
	}
	return clientURL, hubNatsServer, certBundle, serverCfg, err
}
