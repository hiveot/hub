package hiveotsseclient

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hiveot/hub/messaging"
	"github.com/hiveot/hub/messaging/clients/httpclient"
	"github.com/hiveot/hub/messaging/servers/hiveotsseserver"
	jsoniter "github.com/json-iterator/go"
)

// HiveotSseClient is the http/2 client for connecting a WoT client to a
// WoT server using the HiveOT http and sse sub-protocol.
// This implements the IClientConnection interface.
//
// This can be used by both consumers and agents.
// This is intended to be used together with an SSE return channel.
//
// The Forms needed to invoke an operations are obtained using the 'getForm'
// callback, which can be tied to a store of TD documents. The form contains the
// hiveot RequestMessage and ResponseMessage endpoints. If no form is available
// then use the default hiveot endpoints that are defined with this protocol binding.
type HiveotSseClient struct {
	httpclient.HttpBasicClient

	// The full server's base URL sse://host:port/path
	//fullURL string
	// The server host:port
	hostPort string

	// the sse connection path
	ssePath              string
	sseRetryOnDisconnect atomic.Bool
	// handler for closing the sse connection
	sseCancelFn context.CancelFunc

	//isConnected atomic.Bool

	// RPC timeout
	//timeout time.Duration

	// sse variables access
	mux sync.RWMutex

	// http2 client for posting messages
	//httpClient *http.Client
	// authentication bearer token if authenticated
	//bearerToken string

	// getForm obtains the form for sending a request or notification
	// if nil, then the hiveot protocol envelope and URL are used as fallback
	//getForm messaging.GetFormHandler

	// custom headers to include in each request
	//headers map[string]string

	lastError atomic.Pointer[error]
}

// ConnectWithToken sets the bearer token to use with requests and establishes
// an SSE connection.
// If a connection exists it is closed first.
func (cc *HiveotSseClient) ConnectWithToken(token string) error {

	// ensure disconnected (note that this resets retryOnDisconnect)
	cc.Disconnect()

	err := cc.SetBearerToken(token)
	if err != nil {
		return err
	}
	// connectSSE will set 'isConnected' on success
	err = cc.ConnectSSE(token)
	if err != nil {
		cc.HttpBasicClient.SetConnected(false)
		return err
	}
	return err
}

// Disconnect from the server
func (cc *HiveotSseClient) Disconnect() {
	slog.Debug("HiveotSseClient.Disconnect",
		slog.String("clientID", cc.GetConnectionInfo().ClientID),
	)
	cc.mux.Lock()
	cb := cc.sseCancelFn
	cc.sseCancelFn = nil
	cc.mux.Unlock()

	// the connection status will update, if changed, through the sse callback
	if cb != nil {
		cb()
	}

	cc.mux.Lock()
	defer cc.mux.Unlock()
	if cc.IsConnected() {
		cc.GetTlsClient().CloseIdleConnections()
	}
}

// SendNotification Agent posts a notification using the hiveot protocol.
//
// This provides a way for sending responses over plain http to the Hub, which
// will forward it to the connected client. The client must use a sse, wss or other
// connection based protocol.
//
// This passes the notification as-its payload.
//
// This posts the JSON-encoded NotificationMessage on the well-known hiveot notification href.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
//
// This returns an error if the notification could not be delivered to the server
func (cc *HiveotSseClient) SendNotification(msg *messaging.NotificationMessage) error {
	outputJSON, _ := jsoniter.MarshalToString(msg)
	_, _, _, err := cc.Send(http.MethodPost,
		hiveotsseserver.DefaultHiveotPostNotificationHRef, []byte(outputJSON))
	if err != nil {
		slog.Warn("SendNotification failed",
			"clientID", cc.GetConnectionInfo().ClientID,
			"err", err.Error())
	}
	return err
}

// SendRequest sends the RequestMessage envelope to the server, which the SSE server
// listens on.
func (cc *HiveotSseClient) SendRequest(req *messaging.RequestMessage) error {
	outputJSON, _ := jsoniter.MarshalToString(req)
	outputRaw, headers, code, err := cc.Send(http.MethodPost,
		hiveotsseserver.DefaultHiveotPostRequestHRef, []byte(outputJSON))
	_ = headers

	if code == http.StatusOK {
		resp := req.CreateResponse(nil, nil)
		// unmarshal output. This is either the json encoded output or the ResponseMessage envelope
		if outputRaw == nil || len(outputRaw) == 0 {
			// nothing to unmarshal
		} else {
			err = jsoniter.UnmarshalFromString(string(outputRaw), resp)
		}
		if err != nil {
			resp.Error = err.Error()
		}
		// pass a direct response to the application handler
		h := cc.GetAppResponseHandler()
		if h == nil {
			slog.Error("AppResponseHandler is nil")
		} else {
			go func() {
				_ = h(resp)
			}()
		}
	} else if code > 200 && code < 300 {
		// http servers/things might respond with 201 for pending as per spec
		// pending results are passed as a notification.
		var notif *messaging.NotificationMessage
		if outputRaw == nil || len(outputRaw) == 0 {
			// no response yet. do not send process a notification
		} else {
			// pass the notification to the application handler
			notif = req.CreateNotification()
			err = jsoniter.Unmarshal(outputRaw, notif)
			h := cc.GetAppNotificationHandler()
			if h == nil {
				slog.Error("AppNotificationHandler is nil")
			} else {
				go func() {
					h(notif)
				}()
			}
		}
	} else {
		// error response
		resp := req.CreateResponse(nil, nil)
		httpProblemDetail := map[string]string{}
		if outputRaw != nil && len(outputRaw) > 0 {
			err = jsoniter.Unmarshal(outputRaw, &httpProblemDetail)
			resp.Error = httpProblemDetail["title"]
			resp.Value = httpProblemDetail["detail"]
		} else {
			resp.Error = "request failed"
		}
		// pass a direct response to the application handler
		h := cc.GetAppResponseHandler()
		if h == nil {
			slog.Error("AppResponseHandler is nil")
		} else {
			go func() {
				_ = h(resp)
			}()
		}
	}
	return err
}

// SendResponse Agent posts a response using the hiveot protocol.
//
// This provides a way for sending responses over plain http to the Hub, which
// will forward it to the connected client. The client must use a sse, wss or other
// connection based protocol.
//
// This passes the ResponseMessage as its payload as this isn't defined in WoT.
//
// This posts the JSON-encoded ResponseMessage on the well-known hiveot response href.
// In WoT Agents are typically a server, not a client, so this is intended for
// agents that use connection-reversal.
func (cc *HiveotSseClient) SendResponse(resp *messaging.ResponseMessage) error {
	outputJSON, _ := jsoniter.MarshalToString(resp)
	_, _, _, err := cc.Send(http.MethodPost,
		hiveotsseserver.DefaultHiveotPostResponseHRef, []byte(outputJSON))
	return err
}

// NewHiveotSseClient creates a new instance of the http-basic protocol binding client.
// This uses TD forms to perform an operation.
//
//	sseURL of the http and sse server to connect to, including the schema
//	clientID to identify as. Must match the auth token
//	clientCert optional client certificate to connect with
//	caCert of the server to validate the server or nil to not check the server cert
//	getForm is the handler for return a form for invoking an operation. nil for default
//	timeout for waiting for response. 0 to use the default.
func NewHiveotSseClient(
	sseURL string, clientID string, clientCert *tls.Certificate, caCert *x509.Certificate,
	getForm messaging.GetFormHandler, timeout time.Duration) *HiveotSseClient {

	urlParts, err := url.Parse(sseURL)
	if err != nil {
		slog.Error("Invalid URL")
		return nil
	}
	hostPort := urlParts.Host
	ssePath := urlParts.Path

	//cinfo := messaging.ConnectionInfo{
	//	//CaCert:       caCert,
	//	//ClientID:     clientID,
	//	//ConnectionID: "http-" + shortid.MustGenerate(),
	//	//ConnectURL:   sseURL,
	//	ProtocolType: messaging.ProtocolTypeHiveotSSE,
	//	//Timeout:      timeout,
	//}
	cl := HiveotSseClient{
		HttpBasicClient: *httpclient.NewHttpBasicClient(
			sseURL, clientID, clientCert, caCert, getForm, timeout),
		//cinfo: cinfo,
		//clientID: clientID,
		//caCert:   caCert,
		//cid:      "http-" + shortid.MustGenerate(),
		//fullURL:  sseURL,
		ssePath:  ssePath,
		hostPort: hostPort,
		//timeout:  timeout,
	}
	return &cl
}
