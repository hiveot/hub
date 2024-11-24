package wssbinding

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/gorilla/websocket"
	"log/slog"
	"net"
	"net/http"
	"sync/atomic"
)

// ConnectWSS establishes a websocket session with a server
func ConnectWSS(
	clientID string, wssURL string, bearerToken string, caCert *x509.Certificate,
	onConnect func(bool, error),
	onMessage func(jsonMsg string),
) (cancelFn func(), conn *websocket.Conn, err error) {
	var clientCertList []tls.Certificate

	// separate client with a long timeout for sse
	// use a new http client instance to set an indefinite timeout for the sse connection
	//httpClient := tlsclient.NewHttp2TLSClient(caCert, nil, 0)

	slog.Info("ConnectWSS (to hub) - establishing Websocket connection to server",
		slog.String("URL", wssURL),
		slog.String("clientID", clientID),
	)

	// use context to disconnect the client
	wssCtx, wssCancelFn := context.WithCancel(context.Background())
	_ = wssCtx

	// the CA certificate is set in NewTLSClient
	caCertPool := x509.NewCertPool()
	if caCert != nil {
		caCertPool.AddCert(caCert)
	}
	//if clientCert != nil {
	//	clientCertList = []tls.Certificate{*clientCert}
	//}
	//wssURLParsed, _ := url.Parse(wssURL)
	tlsConfig := &tls.Config{
		//ServerName:         wssURLParsed.Hostname(),
		RootCAs: caCertPool,
		//InsecureSkipVerify: caCert == nil,
		InsecureSkipVerify: true,
		Certificates:       clientCertList,
	}

	wssHeader := http.Header{}
	wssHeader.Add("Authorization", "bearer "+bearerToken)
	//parts, _ := url.Parse(wssURL)
	//origin := fmt.Sprintf("%s://%s", parts.Scheme, parts.Host)
	//opts.HTTPHeader.Add("Origin", origin)

	dialer := websocket.DefaultDialer
	dialer.TLSClientConfig = tlsConfig
	dialer.NetDialTLSContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		netConn, err := net.Dial(network, addr)
		if err != nil {
			return nil, err
		}

		// 'NetDialTLSContext' also gets called during the proxy CONNECT for some reason (at this point 'network' equals "TCP" and 'addr' equals "127.0.0.1:8888")
		// The HTTP proxy doesn't support HTTPS however, so I return the established TCP connection early.
		// If I don't do this check, the connection hangs forever (tested with several proxies).
		// This feels kinda hacky though, not sure if this is the correct approach...
		//if p.Host == addr {
		//	return netConn, err
		//}

		// Example TLS handshake
		tlsConn := tls.Client(netConn, tlsConfig)
		if err = tlsConn.Handshake(); err != nil {
			return nil, err
		}

		return tlsConn, nil
	}
	// FIXME: use http/2
	wssConn, _, err := dialer.Dial(wssURL, wssHeader)
	if err != nil {
		wssCancelFn()
		return nil, nil, err
	}

	closeWSSFn := func() {
		err = wssConn.Close()

		// is this needed after close above?
		wssCancelFn()
	}
	// notify the world we're connected
	if onConnect != nil {
		onConnect(true, nil)
	}
	// last, start handling incoming messages
	go func() {
		WSSReadLoop(wssCtx, wssConn, onMessage)
		if onConnect != nil {
			onConnect(false, nil)
		}
	}()

	return closeWSSFn, wssConn, nil
}

// WSSReadLoop reads incoming websocket messages in a loop, until connection closes or context is cancelled
func WSSReadLoop(ctx context.Context, wssConn *websocket.Conn, onMessage func(jsonMsg string)) {

	var readLoop atomic.Bool
	readLoop.Store(true)

	// close the client when the context ends drops
	go func() {
		select {
		case <-ctx.Done(): // remote client connection closed
			slog.Debug("WSSReadLoop: Remote client disconnected")
			// close channel when no-one is writing
			// in the meantime keep reading to prevent deadlock
			_ = wssConn.Close()
			readLoop.Store(false)
		}
	}()

	// read messages from the client until the connection closes
	for readLoop.Load() { // sseMsg := range sseChan {
		_, jsonMsg, err := wssConn.ReadMessage()
		if err != nil {
			// avoid further writes
			readLoop.Store(false)
			// ending the read loop and returning will close the connection
			break
		}
		onMessage(string(jsonMsg))
	}

}
