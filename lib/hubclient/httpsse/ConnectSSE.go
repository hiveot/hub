package httpsse

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
	"net/url"
	"time"
)

//func (cl *HttpSSEClient) ConnectR3SSE(sseURL string, bearerToken string) error {
//	r3c := r3sse.NewClient(sseURL)
//	r3c.Headers["authorization"] = "bearer " + bearerToken
//	parts, _ := url.Parse(sseURL)
//	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
//	r3c.Headers["Origin"] = origin
//	r3c.Connection.Transport = &http.Transport{
//		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
//	}
//	r3c.ReconnectStrategy = backoff.NewConstantBackOff(time.Second)
//
//	go func() {
//		// reconnect doesn't work on server restart.
//		// See also: https://github.com/r3labs/sse/issues/147
//		err := r3c.Subscribe("action", func(msg *r3sse.Event) {
//			se := sse.Event{
//				Type: string(msg.Event),
//				Data: string(msg.Data),
//			}
//			cl.handleSSEEvent(se)
//		})
//		_ = err
//		slog.Info("r3c subscribe stopped")
//	}()
//	return nil
//}

// ConnectSSE establishes a sse session over the Hub HTTPS connection.
// All hub messages are send as type ThingMessage, containing thingID, key, payload and sender
func (cl *HttpSSEClient) ConnectSSE(sseURL string, bearerToken string, httpClient *http.Client) error {

	// just testing
	//return cl.ConnectR3SSE(sseURL, bearerToken)

	slog.Info("ConnectSSE", slog.String("sseURL", sseURL))

	//sseURL := fmt.Sprintf("https://%s%s", cl.hostPort, cl.ssePath)
	//req, err := tlsclient.NewRequest("GET", sseURL, bearerToken, []byte{})
	//if err != nil {
	//	return err
	//}

	// use context to disconnect the client
	sseCtx, sseCancelFn := context.WithCancel(context.Background())
	bodyReader := bytes.NewReader([]byte{})
	req, err := http.NewRequestWithContext(sseCtx, http.MethodGet, sseURL, bodyReader)
	if err != nil {
		sseCancelFn()
		return err
	}
	req.Header.Add("Authorization", "bearer "+bearerToken)
	parts, _ := url.Parse(sseURL)
	origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	req.Header.Add("Origin", origin)
	//req.Header.Add("Upgrade", "h2")
	//req.Close = false

	cl.sseCancelFn = sseCancelFn
	//req = req.WithContext(sseCtx)

	//sse client upgrade will set this to stream
	//req.Header.Set("Content-Type", "application/json")
	sseClient := sse.Client{
		HTTPClient: httpClient,
		OnRetry: func(err error, _ time.Duration) {
			slog.Info("SSE Connection retry", "err", err)
		},
	}
	conn := sseClient.NewConnection(req)

	// increase buffer size to 1M
	// TODO: make limit configurable
	//https://github.com/tmaxmax/go-sse/issues/32
	newBuf := make([]byte, 0, 1024*65)
	conn.Buffer(newBuf, cl._maxSSEMessageSize)

	//conn.Parser.Buffer = make([]byte, 1000000) // test 1MB buffer

	remover := conn.SubscribeToAll(cl.handleSSEEvent)
	go func() {
		// connect and report an error if connection ends due to reason other than context cancelled
		err := conn.Connect()
		if err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("SSE connection failed (server shutdown or connection interrupted)",
				"clientID", cl._status.ClientID,
				"err", err.Error())
		}
		remover()
	}()
	return nil
}
