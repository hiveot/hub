package testenv

import (
	"github.com/hiveot/hub/api/go/auth"
	"github.com/hiveot/hub/api/go/msgserver"
	"github.com/hiveot/hub/core/natsmsgserver"
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

// test users and devices
var TestClients = []msgserver.ClientAuthInfo{
	{
		ClientID:   TestDevice1ID,
		ClientType: auth.ClientTypeDevice,
		PubKey:     TestDevice1Pub,
		Role:       auth.ClientRoleNone,
	},
	{
		ClientID:     TestUser1ID,
		ClientType:   auth.ClientTypeUser,
		PasswordHash: string(TestUser1bcrypt),
		Role:         auth.ClientRoleViewer,
	},
	{
		ClientID:   TestUser2ID,
		ClientType: auth.ClientTypeUser,
		PubKey:     TestUser2Pub,
		Role:       auth.ClientRoleOperator,
	},
	{
		ClientID:   TestService1ID,
		ClientType: auth.ClientTypeService,
		PubKey:     TestService1Pub,
		Role:       auth.ClientRoleAdmin,
	},
}

// StartNatsTestServer generate a test configuration and starts a NKeys based nats test server
// A new temporary storage directory is used.
func StartNatsTestServer() (
	clientURL string,
	hubNatsServer *natsmsgserver.NatsMsgServer,
	certBundle certs.TestCertBundle,
	config *natsmsgserver.NatsServerConfig, err error) {

	certBundle = certs.CreateTestCertBundle()

	serverCfg := &natsmsgserver.NatsServerConfig{
		Port:   9990,
		CaCert: certBundle.CaCert,
		CaKey:  certBundle.CaKey,
		//ServerCert: certBundle.ServerCert, // auto generate
		//Debug: true,
	}
	//
	tmpDir := path.Join(os.TempDir(), "nats-testserver")
	_ = os.RemoveAll(tmpDir)
	err = serverCfg.Setup(tmpDir, tmpDir, false)
	if err == nil {
		hubNatsServer = natsmsgserver.NewNatsMsgServer(serverCfg, auth.DefaultRolePermissions)
		clientURL, err = hubNatsServer.Start()
	}
	return clientURL, hubNatsServer, certBundle, serverCfg, err
}
