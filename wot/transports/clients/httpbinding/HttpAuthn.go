package httpbinding

import (
	"encoding/json"
	"fmt"
	jsoniter "github.com/json-iterator/go"
	"github.com/teris-io/shortid"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
)

// HiveoT authentication methods supported by the binding.
//
// Paths used by this protocol binding - keep in sync with the server paths.
//
// THIS WILL BE REMOVED AFTER THE PROTOCOL BINDING PUBLISHES THESE IN THE TDD.
// The hub client will need the TD (ConsumedThing) to determine the paths.
const (

	// deprecated authn service - use the generated constants or forms
	PostLoginPath = "/authn/login"
	// deprecated authn service - use the generated constants
	PostLogoutPath = "/authn/logout"
	// deprecated authn service - use the generated constants
	PostRefreshPath = "/authn/refresh"
)

// ConnectWithLoginForm invokes login using a form - temporary helper
// intended for testing a connection with a web server.
//
// This sets the bearer token for further requests
func (cl *HttpTransportClient) ConnectWithLoginForm(password string) error {
	formMock := url.Values{}
	formMock.Add("loginID", cl.GetClientID())
	formMock.Add("password", password)
	fullURL := fmt.Sprintf("https://%s/login", cl.BaseHostPort)

	//PostForm should return a cookie that should be used in the http connection
	resp, err := cl.httpClient.PostForm(fullURL, formMock)
	if err == nil {
		// get the session token from the cookie
		cookie := resp.Request.Header.Get("cookie")
		kvList := strings.Split(cookie, ",")

		for _, kv := range kvList {
			kvParts := strings.SplitN(kv, "=", 2)
			if kvParts[0] == "session" {
				cl.bearerToken = kvParts[1]
				break
			}
		}
	}
	return err
}

// ConnectWithPassword connects to the Hub TLS server using a login ID and password
// and obtain an auth token for use with ConnectWithToken.
//
// This is currently hub specific, until a standard way is fond using the Hub TD
func (cl *HttpTransportClient) ConnectWithPassword(password string) (newToken string, err error) {

	slog.Info("ConnectWithPassword",
		"clientID", cl.GetClientID(), "connectionID", cl.GetConnectionID())

	// FIXME: figure out how a standard login method is used to obtain an auth token
	loginMessage := map[string]string{
		"login":    cl.GetClientID(),
		"password": password,
	}
	argsJSON, _ := json.Marshal(loginMessage)
	requestID := shortid.MustGenerate()
	resp, _, err := cl._send(
		http.MethodPost, PostLoginPath, "", "", "", argsJSON, requestID)
	if err != nil {
		slog.Warn("ConnectWithPassword failed", "err", err.Error())
		return "", err
	}
	token := ""
	err = cl.Unmarshal(resp, &token)
	if err != nil {
		err = fmt.Errorf("ConnectWithPassword: unexpected response: %s", err)
		return "", err
	}
	// store the bearer token further requests
	cl.BaseMux.Lock()
	cl.bearerToken = token
	cl.BaseMux.Unlock()
	cl.BaseIsConnected.Store(true)

	return token, err
}

// ConnectWithToken sets the authentication bearer token to authenticate http requests.
func (cl *HttpTransportClient) ConnectWithToken(token string) (newToken string, err error) {
	cl.BaseMux.Lock()
	cl.bearerToken = token
	cl.BaseMux.Unlock()
	cl.BaseIsConnected.Store(true)

	newToken, err = cl.RefreshToken(token)
	return newToken, err
}

// RefreshToken refreshes the authentication token
// The resulting token can be used with 'ConnectWithToken'
// This is specific to the Hiveot Hub.
func (cl *HttpTransportClient) RefreshToken(oldToken string) (newToken string, err error) {

	// FIXME: what is the standard for refreshing a token using http?
	slog.Info("RefreshToken",
		slog.String("clientID", cl.GetClientID()))

	// the bearer token holds the old token
	payload, _ := jsoniter.Marshal(oldToken)
	resp, _, err := cl._send(
		"POST", PostRefreshPath, "", "", "", payload, "")

	// set the new token as the bearer token
	if err == nil {
		err = jsoniter.Unmarshal(resp, &newToken)

		if err == nil {
			// reconnect using the new token
			cl.BaseMux.Lock()
			cl.bearerToken = newToken
			cl.BaseMux.Unlock()
		}
	}
	return newToken, err
}

// Logout from the server and end the session.
// This is specific to the Hiveot Hub.
func (cl *HttpTransportClient) Logout() error {
	// TODO: can this be derived from a form?
	slog.Info("Logout",
		slog.String("clientID", cl.GetClientID()))
	_, _, err := cl._send("POST", PostLogoutPath, "", "", "", nil, "")
	return err
}
