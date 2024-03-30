package tlsserver_test

import (
	"fmt"
	"github.com/hiveot/hub/lib/certs"
	"github.com/hiveot/hub/lib/tlsclient"
	"github.com/hiveot/hub/runtime/tlsserver"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var serverAddress string
var serverPort uint = 4444
var clientHostPort string
var testCerts certs.TestCertBundle

// TestMain runs a http server
// Used for all test cases in this package
func TestMain(m *testing.M) {
	slog.Info("------ TestMain of TLSServer_test.go ------")
	// serverAddress = hubnet.GetOutboundIP("").String()
	// use the localhost interface for testing
	serverAddress = "127.0.0.1"
	// hostnames := []string{serverAddress}
	clientHostPort = fmt.Sprintf("%s:%d", serverAddress, serverPort)

	testCerts = certs.CreateTestCertBundle()
	res := m.Run()

	time.Sleep(time.Second)
	os.Exit(res)
}

func TestStartStop(t *testing.T) {
	srv, router := tlsserver.NewTLSServer(serverAddress, serverPort,
		testCerts.ServerCert, testCerts.CaCert)
	_ = router
	err := srv.Start()
	assert.NoError(t, err)
	srv.Stop()
}

func TestNoServerCert(t *testing.T) {
	srv, router := tlsserver.NewTLSServer(serverAddress, serverPort,
		nil, testCerts.CaCert)
	_ = router
	err := srv.Start()
	assert.Error(t, err)
	srv.Stop()
}

// connect without authentication
func TestNoAuth(t *testing.T) {
	path1 := "/hello"
	path1Hit := 0
	srv, router := tlsserver.NewTLSServer(serverAddress, serverPort,
		testCerts.ServerCert, testCerts.CaCert)
	_ = router
	router.Get(path1, func(w http.ResponseWriter, req *http.Request) {
		slog.Info("TestNoAuth: path1 hit")
		path1Hit++
	})

	err := srv.Start()
	assert.NoError(t, err)

	cl := tlsclient.NewTLSClient(clientHostPort, nil)
	require.NoError(t, err)
	cl.ConnectNoAuth()
	_, err = cl.Get(path1)
	assert.NoError(t, err)
	assert.Equal(t, 1, path1Hit)

	cl.Close()
	srv.Stop()
}

// Test with invalid login authentication
//func TestUnauthorized(t *testing.T) {
//	path1 := "/test1"
//	//loginID1 := "user1"
//	//password1 := "user1pass"
//
//	// setup server and client environment
//	srv, router := tlsserver.NewTLSServer(serverAddress, serverPort,
//		testCerts.ServerCert, testCerts.CaCert)
//
//	err := srv.Start()
//	assert.NoError(t, err)
//	//
//	jwtAuthorizer := tlsserver.NewJWTAuthenticator()
//	router.Use(jwtAuthorizer)
//	srv.AddHandler(path1, func(string, http.ResponseWriter, *http.Request) {
//		slog.Info("TestNoAuth: path1 hit")
//		assert.Fail(t, "did not expect the request to pass")
//	})
//	//
//	cl := tlsclient.NewTLSClient(clientHostPort, testCerts.CaCert)
//	assert.NoError(t, err)
//
//	// AuthMethodNone creates a client without any authentication method
//	cl.ConnectNoAuth()
//
//	// ... which causes any request to fail
//	_, err = cl.Get(path1)
//	assert.Error(t, err)
//
//	cl.Close()
//	srv.Stop()
//}

//func TestCertAuth(t *testing.T) {
//	path1 := "/hello"
//	path1Hit := 0
//	srv, router := tlsserver.NewTLSServer(serverAddress, serverPort,
//		testCerts.ServerCert, testCerts.CaCert)
//	err := srv.Start()
//	assert.NoError(t, err)
//	// handler can be added any time
//	srv.AddHandler(path1, func(string, http.ResponseWriter, *http.Request) {
//		slog.Info("TestAuthCert: path1 hit")
//		path1Hit++
//	})
//
//	cl := tlsclient.NewTLSClient(clientHostPort, testCerts.CaCert)
//	require.NoError(t, err)
//	err = cl.ConnectWithClientCert(testCerts.ClientCert)
//	assert.NoError(t, err)
//	_, err = cl.Get(path1)
//	assert.NoError(t, err)
//	assert.Equal(t, 1, path1Hit)
//
//	cl.Close()
//	srv.Stop()
//}

// Test valid authentication using JWT
//func TestQueryParams(t *testing.T) {
//	path2 := "/hello"
//	path2Hit := 0
//	srv, router := tlsserver.NewTLSServer(serverAddress, serverPort,
//		testCerts.ServerCert, testCerts.CaCert)
//	err := srv.Start()
//	assert.NoError(t, err)
//	srv.AddHandler(path2, func(userID string, resp http.ResponseWriter, req *http.Request) {
//		// query string
//		q1 := srv.GetQueryString(req, "query1", "")
//		assert.Equal(t, "bob", q1)
//		// fail not a number
//		_, err := srv.GetQueryInt(req, "query1", 0) // not a number
//		assert.Error(t, err)
//		// query of number
//		q2, _ := srv.GetQueryInt(req, "query2", 0)
//		assert.Equal(t, 3, q2)
//		// default should work
//		q3 := srv.GetQueryString(req, "query3", "default")
//		assert.Equal(t, "default", q3)
//		// multiple parameters fail
//		_, err = srv.GetQueryInt(req, "multi", 0)
//		assert.Error(t, err)
//		path2Hit++
//	})
//
//	cl := tlsclient.NewTLSClient(clientHostPort, testCerts.CaCert)
//	require.NoError(t, err)
//	err = cl.ConnectWithClientCert(testCerts.ClientCert)
//	assert.NoError(t, err)
//
//	_, err = cl.Get(fmt.Sprintf("%s?query1=bob&query2=3&multi=a&multi=b", path2))
//	assert.NoError(t, err)
//	assert.Equal(t, 1, path2Hit)
//
//	cl.Close()
//	srv.Stop()
//}

func TestWriteResponse(t *testing.T) {
	path2 := "/hello"
	message := "hello world"
	path2Hit := 0
	srv, router := tlsserver.NewTLSServer(serverAddress, serverPort,
		testCerts.ServerCert, testCerts.CaCert)
	err := srv.Start()
	assert.NoError(t, err)
	router.Get(path2, func(w http.ResponseWriter, req *http.Request) {
		_, _ = w.Write([]byte(message))
		w.WriteHeader(http.StatusOK)
		//srv.WriteBadRequest(resp, "bad request")
		//srv.WriteInternalError(resp, "internal error")
		//srv.WriteNotFound(resp, "not found")
		//srv.WriteNotImplemented(resp, "not implemented")
		//srv.WriteUnauthorized(resp, "unauthorized")
		path2Hit++
	})

	cl := tlsclient.NewTLSClient(clientHostPort, testCerts.CaCert)
	require.NoError(t, err)
	err = cl.ConnectWithClientCert(testCerts.ClientCert)
	assert.NoError(t, err)

	reply, err := cl.Get(path2)
	assert.NoError(t, err)
	assert.Equal(t, 1, path2Hit)
	assert.Equal(t, message, string(reply))

	cl.Close()
	srv.Stop()
}

func TestBadPort(t *testing.T) {
	srv, router := tlsserver.NewTLSServer(serverAddress, 1, // bad port
		testCerts.ServerCert, testCerts.CaCert)
	_ = router
	err := srv.Start()
	assert.Error(t, err)
}

//
//// Test BASIC authentication
//func TestBasicAuth(t *testing.T) {
//	path1 := "/test1"
//	path1Hit := 0
//	loginID1 := "user1"
//	password1 := "user1pass"
//
//	// setup server and client environment
//	srv := tlsserver.NewTLSServer(serverAddress, serverPort,
//		testCerts.ServerCert, testCerts.CaCert)
//	srv.EnableBasicAuth(func(userID, password string) bool {
//		path1Hit++
//		return userID == loginID1 && password == password1
//	})
//	err := srv.Start()
//	assert.NoError(t, err)
//	//
//	srv.AddHandler(path1, func(string, http.ResponseWriter, *http.Request) {
//		slog.Info("TestBasicAuth: path1 hit")
//		path1Hit++
//	})
//	//
//	cl := tlsclient.NewTLSClient(clientHostPort, testCerts.CaCert)
//	assert.NoError(t, err)
//	cl.ConnectWithBasicAuth(loginID1, password1)
//
//	// test the auth with a GET request
//	_, err = cl.Get(path1)
//	assert.NoError(t, err)
//	assert.Equal(t, 2, path1Hit)
//
//	// test a failed login
//	cl.Close()
//	cl.ConnectWithBasicAuth(loginID1, "wrongpassword")
//	_, err = cl.Get(path1)
//	assert.Error(t, err)
//	assert.Equal(t, 3, path1Hit) // should not increase
//
//	cl.Close()
//	srv.Stop()
//}
