package httpclient

import (
	"errors"
	"fmt"
	"github.com/hiveot/hub/transports/servers/httpserver"
	"github.com/hiveot/hub/wot"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// ConnectWithLoginForm invokes login using a form - temporary helper
// intended for testing a connection with a web server.
//
// This sets the bearer token for further requests. It requires the server
// to set a session cookie in response to the login.
func (cl *HttpConsumerClient) ConnectWithLoginForm(
	password string) (newToken string, err error) {

	// FIXME: does this client need a cookie jar???

	formMock := url.Values{}
	formMock.Add("loginID", cl.GetClientID())
	formMock.Add("password", password)

	var loginHRef string
	f := cl.BaseGetForm(wot.HTOpLoginWithForm, "", "")
	if f != nil {
		loginHRef, _ = f.GetHRef()
	}
	loginURL, err := url.Parse(loginHRef)
	if err != nil {
		return "", err
	}
	if loginURL.Host == "" {
		loginHRef = cl.BaseFullURL + loginHRef
	}

	//PostForm should return a cookie that should be used in the http connection
	if loginHRef == "" {
		return "", errors.New("Login path not found in getForm")
	}
	resp, err := cl.httpClient.PostForm(loginHRef, formMock)
	if err != nil {
		return "", err
	}

	// get the session token from the cookie
	//cookie := resp.Request.Header.Get("cookie")
	cookie := resp.Header.Get("cookie")
	kvList := strings.Split(cookie, ",")

	for _, kv := range kvList {
		kvParts := strings.SplitN(kv, "=", 2)
		if kvParts[0] == "session" {
			cl.bearerToken = kvParts[1]
			break
		}
	}
	if cl.bearerToken == "" {
		slog.Error("No session cookie was received on login")
	}
	return cl.bearerToken, err
}

// ConnectWithPassword connects to the TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *HttpConsumerClient) ConnectWithPassword(password string) (newToken string, err error) {

	slog.Info("ConnectWithPassword",
		"clientID", cl.GetClientID(), "connectionID", cl.GetConnectionID())

	// FIXME: figure out how a standard login method is used to obtain an auth token
	loginMessage := map[string]string{
		"login":    cl.GetClientID(),
		"password": password,
	}
	f := cl.BaseGetForm(wot.HTOpLogin, "", "")
	if f == nil {
		err = fmt.Errorf("missing form for login operation")
		slog.Error(err.Error())
		return "", err
	}
	method, _ := f.GetMethodName()
	href, _ := f.GetHRef()

	dataJSON := cl.Marshal(loginMessage)
	outputRaw, _, err := cl._send(method, href, dataJSON, "")

	if err != nil {
		slog.Warn("ConnectWithPassword failed", "err", err.Error())
		return "", err
	}
	err = jsoniter.Unmarshal(outputRaw, &newToken)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: unexpected response: %s", err)
		return "", err
	}

	// store the bearer token further requests
	cl.BaseMux.Lock()
	cl.bearerToken = newToken
	cl.BaseMux.Unlock()
	//cl.BaseIsConnected.Store(true)

	return newToken, err
}

// ConnectWithToken sets the authentication bearer token to authenticate http requests.
func (cl *HttpConsumerClient) ConnectWithToken(token string) (newToken string, err error) {
	cl.BaseMux.Lock()
	cl.bearerToken = token
	cl.BaseMux.Unlock()
	//cl.BaseIsConnected.Store(true)

	newToken, err = cl.RefreshToken(token)
	return newToken, err
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *HttpConsumerClient) RefreshToken(oldToken string) (newToken string, err error) {

	newToken, err = cl.BaseClient.RefreshToken(oldToken)
	if err == nil {
		cl.BaseMux.Lock()
		cl.bearerToken = newToken
		cl.BaseMux.Unlock()
	}
	return newToken, err
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *HttpConsumerClient) Logout() error {
	// TODO: can this be derived from a form?
	slog.Info("Logout",
		slog.String("clientID", cl.GetClientID()))
	_, _, err := cl._send(http.MethodPost, httpserver.HttpPostLogoutPath, nil, "")
	return err
}
