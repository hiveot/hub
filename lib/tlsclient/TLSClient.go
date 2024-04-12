// Package tlsclient with a TLS client helper supporting certificate, JWT or Basic authentication
package tlsclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"golang.org/x/net/publicsuffix"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"time"
)

// Authentication methods for use with ConnectWithLoginID
// Use AuthMethodDefault unless there is a good reason not to
const (
	AuthMethodBasic  = "basic"  // basic auth for backwards compatibility when connecting
	AuthMethodDigest = "digest" // digest auth for backwards compatibility when connecting
	AuthMethodNone   = ""       // disable authentication, for testing
	AuthMethodJwt    = "jwt"    // JSON web token for use with WoST server (default)
)

// standardized query parameter names for querying servers
const (
	// ParamOffset offset in case of multiple requests
	ParamOffset = "offset"
	// ParamLimit contains maximum number of results
	ParamLimit = "limit"
	// ParamQuery contains a query
	ParamQuery = "queryparams"
	// ParamUpdatedSince contains a ISO8601 datetime
	ParamUpdatedSince = "updatedSince"
	// ParamThings contains a list of Thing IDs to query for
	ParamThings = "things"
)

// JwtAuthLogin defines the login request message to sent when using JWT authentication
type JwtAuthLogin struct {
	LoginID    string `json:"login"` // typically the email
	Password   string `json:"password"`
	RememberMe bool   `json:"rememberMe,omitempty"` // store refresh token in cookie
}

// TLSClient is a simple TLS Client with authentication using certificates or JWT authentication with login/pw
type TLSClient struct {
	// host and port of the server to connect to
	hostPort        string
	caCert          *x509.Certificate
	caCertPool      *x509.CertPool
	httpClient      *http.Client
	timeout         time.Duration
	checkServerCert bool

	// client certificate mutual authentication
	clientCert *tls.Certificate

	// User ID for authentication
	userID string

	// JWT bearer token after login, refresh, or external source
	// Invoke will use this if set.
	bearerToken string
}

// Certificate returns the client auth certificate or nil if none is used
func (cl *TLSClient) Certificate() *tls.Certificate {
	return cl.clientCert
}

// Close the connection with the server
func (cl *TLSClient) Close() {
	slog.Info("TLSClient.Close: Closing client connection")

	if cl.httpClient != nil {
		cl.httpClient.CloseIdleConnections()
		cl.httpClient = nil
	}
}

// connect sets-up the http client with TLS transport
func (cl *TLSClient) connect() *http.Client {
	// the CA certificate is set in NewTLSClient
	tlsConfig := &tls.Config{
		RootCAs:            cl.caCertPool,
		InsecureSkipVerify: !cl.checkServerCert,
	}

	tlsTransport := http.DefaultTransport
	tlsTransport.(*http.Transport).TLSClientConfig = tlsConfig

	// FIXME:
	// 1 does this work if the server is connected using an IP address?
	// 2. How are cookies stored between sessions?
	cjarOpts := &cookiejar.Options{PublicSuffixList: publicsuffix.List}
	cjar, err := cookiejar.New(cjarOpts)
	if err != nil {
		err = fmt.Errorf("NewTLSClient: error setting cookiejar. The use of bearer tokens might not work: %w", err)
		slog.Error(err.Error())
	}

	return &http.Client{
		Transport: tlsTransport,
		Timeout:   cl.timeout,
		Jar:       cjar,
	}
}

// ConnectNoAuth creates a connection with the server without client authentication
// Only requests that do not require authentication will succeed
func (cl *TLSClient) ConnectNoAuth() {
	cl.httpClient = cl.connect()
}

// ConnectWithClientCert creates a connection with the server using a client certificate for mutual authentication.
// The provided certificate must be signed by the server's CA.
//
//	clientCert client tls certificate containing x509 cert and private key
//
// Returns nil if successful, or an error if connection failed
func (cl *TLSClient) ConnectWithClientCert(clientCert *tls.Certificate) (err error) {
	var clientCertList = make([]tls.Certificate, 0)

	if clientCert == nil {
		err = fmt.Errorf("TLSClient.ConnectWithClientCert, No client key/certificate provided")
		slog.Error(err.Error())
		return err
	}

	// test if the given cert is valid for our CA
	if cl.caCert != nil {
		opts := x509.VerifyOptions{
			Roots:     cl.caCertPool,
			KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		}
		x509Cert, err := x509.ParseCertificate(clientCert.Certificate[0])
		if err == nil {
			// FIXME: TestCertAuth: certificate specifies incompatible key usage
			// why? Is the certpool invalid? Yet the test succeeds
			_, err = x509Cert.Verify(opts)
		}
		if err != nil {
			err = fmt.Errorf("ConnectWithClientCert: certificate verfication failed: %w", err)
			slog.Error(err.Error())
			return err
		}
	}
	cl.clientCert = clientCert
	clientCertList = append(clientCertList, *clientCert)

	tlsConfig := &tls.Config{
		RootCAs:            cl.caCertPool,
		Certificates:       clientCertList,
		InsecureSkipVerify: !cl.checkServerCert,
	}

	tlsTransport := http.DefaultTransport
	tlsTransport.(*http.Transport).TLSClientConfig = tlsConfig

	cl.httpClient = &http.Client{
		Transport: tlsTransport,
		Timeout:   cl.timeout,
	}
	return nil
}

// ConnectWithToken Sets login ID and secret for JWT authentication using a
// token obtained at login or elsewhere.
//
// No error is returned as this just sets up the token and http client. A messages are send yet.
func (cl *TLSClient) ConnectWithToken(loginID string, token string) {
	cl.userID = loginID
	cl.bearerToken = token

	cl.httpClient = cl.connect()
}

// ConnectWithPassword requests JWT tokens using loginID/password
// If a CA certificate is not available then insecure-skip-verify is used to allow
// connection to an unverified server (leap of faith).
//
// The server returns a JwtAuthResponse message with a jwt bearer token for use in the authorization header.
//
//	loginID username or application ID to identify as.
//	secret to authenticate with.
//
// Returns new auth token if successful or an error if setting up of authentication failed.
func (cl *TLSClient) ConnectWithPassword(loginID string, secret string) (token string, err error) {
	cl.userID = loginID

	loginURL := fmt.Sprintf("https://%s%s", cl.hostPort, vocab.PostLoginPath)

	// create tlsTransport
	cl.httpClient = cl.connect()

	// Authenticate with JWT requires a cookiejar to store the refresh token
	loginMessage := JwtAuthLogin{
		LoginID:  loginID,
		Password: secret,
	}
	// resp, err2 := cl.Post(cl.jwtLoginPath, authLogin)
	resp, err2 := cl.Invoke("POST", loginURL, loginMessage)
	if err2 != nil {
		err = fmt.Errorf("ConnectWithPassword: login to %s failed. %s", loginURL, err2)
		return "", err
	}
	//var jwtResp JwtAuthResponse
	//err2 = json.Unmarshal(resp, &jwtResp)
	//err2 = json.Unmarshal(resp, &token)

	//if err2 != nil {
	//	err = fmt.Errorf("ConnectWithPassword: JWT login to %s has unexpected response message: %s", loginURL, err2)
	//	return "", err
	//}
	//cl.bearerToken = jwtResp.Token
	cl.bearerToken = string(resp)
	return cl.bearerToken, err
}

// Login requests JWT tokens using loginID/password
// If a CA certificate is not available then insecure-skip-verify is used to allow
// connection to an unverified server (leap of faith).
//

// Delete sends a delete message with json payload
//
//	path to invoke
//	msg message object to include. This will be marshalled to json
func (cl *TLSClient) Delete(path string, msg interface{}) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("DELETE", url, msg)
}

// Get is a convenience function to send a request
//
//	path to invoke
func (cl *TLSClient) Get(path string) ([]byte, error) {
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("GET", url, nil)
}

// Invoke a HTTPS method and read response
// If a JWT authentication is enabled then add the bearer token to the header
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	method: GET, PUT, POST, ...
//	url: full URL to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Invoke(method string, url string, msg interface{}) ([]byte, error) {
	var body io.Reader = http.NoBody
	var err error
	var req *http.Request
	contentType := "application/json"

	if cl == nil || cl.httpClient == nil {
		err = fmt.Errorf("Invoke: '%s'. Client is not started", url)
		return nil, err
	}
	slog.Info("TLSClient.Invoke", "method", method, "url", url)

	// careful, a double // in the path causes a 301 and changes post to get
	// url := fmt.Sprintf("https://%s%s", hostPort, path)
	if msg != nil {
		// only marshal to JSON if this isn't a string
		switch msgWithType := msg.(type) {
		case string:
			body = bytes.NewReader([]byte(msgWithType))
		case []byte:
			body = bytes.NewReader(msgWithType)
		default:
			bodyBytes, _ := json.Marshal(msg)
			body = bytes.NewReader(bodyBytes)
		}
	}
	req, err = http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if cl.bearerToken != "" {
		req.Header.Add("Authorization", "bearer "+cl.bearerToken)
	} else {
		// no authentication
	}
	if err != nil {
		return nil, err
	}

	// set headers
	req.Header.Set("Content-Type", contentType)

	resp, err := cl.httpClient.Do(req)
	if err != nil {
		err = fmt.Errorf("Invoke: %s %s: %w", method, url, err)
		return nil, err
	}
	respBody, err := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		msg := fmt.Sprintf("%s: %s", resp.Status, respBody)
		if resp.Status == "" {
			msg = fmt.Sprintf("%d (%s): %s", resp.StatusCode, resp.Status, respBody)
		}
		err = errors.New(msg)
	}
	if err != nil {
		err = fmt.Errorf("Invoke: Error %s %s: %w", method, url, err)
		return nil, err
	}
	return respBody, err
}

// Post a message.
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Post(path string, msg interface{}) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("POST", url, msg)
}

// Put a message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Put(path string, msg interface{}) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("PUT", url, msg)
}

// Patch sends a patch message with json payload
// If msg is a string then it is considered to be already serialized.
// If msg is not a string then it will be json encoded.
//
//	path to invoke
//	msg contains the request body as a string or object
func (cl *TLSClient) Patch(path string, msg interface{}) ([]byte, error) {
	// careful, a double // in the path causes a 301 and changes POST to GET
	url := fmt.Sprintf("https://%s%s", cl.hostPort, path)
	return cl.Invoke("PATCH", url, msg)
}

// RefreshToken refreshes the JWT token
//
// This returns a struct with new access and refresh token
func (cl *TLSClient) RefreshToken(refreshURL string) (newToken string, err error) {
	if refreshURL == "" {
		refreshURL = fmt.Sprintf("https://%s%s", cl.hostPort, vocab.PostRefreshPath)
	}

	// old token exists in client cookie
	req, err := http.NewRequest("POST", refreshURL, http.NoBody)
	var resp *http.Response
	if err != nil {
		err = fmt.Errorf("RefreshToken: Error creating request for URL %s: %w", refreshURL, err)
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = cl.httpClient.Do(req)

	if err != nil {
		err = fmt.Errorf("RefreshToken: Error using URL %s: %w", refreshURL, err)
		return "", err
	} else if resp.StatusCode >= 400 {
		err = fmt.Errorf("RefreshToken: refresh using URL %s failed with: %s", refreshURL, resp.Status)
		return "", err
	}
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		err = fmt.Errorf("RefreshToken: failed with error %w", err)
		return "", err
	}
	err = json.Unmarshal(respBody, &newToken)
	return newToken, err
}

// NewTLSClient creates a new TLS Client instance.
// Use connect/Close to open and close connections
//
//	hostPort is the server hostname or IP address and port to connect to
//	caCert with the x509 CA certificate, nil if not available
//
// returns TLS client for submitting requests
func NewTLSClient(hostPort string, caCert *x509.Certificate) *TLSClient {
	var checkServerCert bool
	caCertPool := x509.NewCertPool()

	// Use CA certificate for server authentication if it exists
	if caCert == nil {
		slog.Info("NewTLSClient: No CA certificate. InsecureSkipVerify used",
			slog.String("destination", hostPort))
		checkServerCert = false
	} else {
		slog.Info("NewTLSClient: CA certificate",
			slog.String("destination", hostPort),
			slog.String("caCert CN", caCert.Subject.CommonName))
		caCertPool.AddCert(caCert)
		checkServerCert = true
	}

	cl := &TLSClient{
		hostPort:        hostPort,
		timeout:         time.Second * 10,
		caCertPool:      caCertPool,
		caCert:          caCert,
		checkServerCert: checkServerCert,
	}

	return cl
}
