package sseserver

import (
	"context"
	"github.com/tmaxmax/go-sse"
	"net/http"
)

// SSEServer supports pushing server messages to the client using the SSE protocol.
// This supports two server implementations, go-sse and the internal ht-sse server.
//
// The go-sse is conformant to the standard but there seem to be a connection handling bug
// when clients reconnect. This is currently under investigation.
//
// The ht-sse server is the internal test server. While it works well it is not fully
// conformant to the sse specification.
type SSEServer struct {
	//htsse *HTSSEServer
	gosse *sse.Server
}

// ServeHttp handles a connection request
func (srv *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if srv.gosse != nil {
		// NOTE! The Content-Type header must be set to text/event-stream otherwise
		// go-sse client will not retry after graceful disconnect.
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Connection", "keep-alive")

		srv.gosse.ServeHTTP(w, r)
	} else {
		HTServeHttp(w, r)
	}
}
func (svc *SSEServer) Stop() {
	if svc.gosse != nil {
		svc.gosse.Shutdown(context.Background())
	}
}

func NewSSEServer() *SSEServer {
	srv := SSEServer{
		// disable gosse to fall back to the built-in test server
		// disabled for now due to race conditions
		//    https://github.com/tmaxmax/go-sse/issues/35
		//gosse: NewGoSSEServer(),
	}
	return &srv
}
