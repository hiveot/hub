package sse

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/runtime/api"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"github.com/tmaxmax/go-sse"
	"log/slog"
	"net/http"
)

// SSEHandler supports pushing server messages to the client using the SSE protocol.
// This handles incoming SSE connections and links them to client sessions using
// the authentication token. Messages can be sent to the clients using the
// session object.
// The token must be a session token issued through login or refresh methods.
type SSEHandler struct {
	sessionAuth api.IAuthenticator
	//handleMessage router.MessageHandler
	gosse *sse.Server
}

// handleSseConnect handles incoming SSE connections, authenticates the client
// Sse requests are refused if no valid session found.
func (svc *SSEHandler) handleSseConnect(w http.ResponseWriter, r *http.Request) {
	// Set headers for SSE response
	//w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Expose-Headers", "Content-Type")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Cache-Control", "private, no-cache, no-store, must-revalidate, max-age=0, no-transform")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Connection", "keep-alive")

	// An active session is required before accepting the request. This is created on
	// authentication/login.
	cs, err := sessions.GetSessionFromContext(r)
	if cs == nil || err != nil {
		slog.Warn("No session available, telling client to delay retry to 30 seconds")

		// set retry to a large number
		// while this doesn't redirect, it does stop it from holding a connection.
		// see https://javascript.info/server-sent-events#reconnection
		//_, _ = fmt.Fprintf(w, "retry: %s\nevent:%s\n\n",
		//	"30000", "logout")
		//// this result code doesn't seem to work
		//w.WriteHeader(http.StatusUnauthorized)
		errMsg := fmt.Sprintf("retry: %s\nevent:%s\n\n",
			"30000", "logout")
		http.Error(w, errMsg, http.StatusUnauthorized)
		w.(http.Flusher).Flush()
		return
	}

	// establish a client event channel for sending messages back to the client
	sseChan := cs.CreateSSEChan()

	// TODO: if this is a first connection of the client send a connected event

	slog.Info("SseHandler. New SSE connection",
		slog.String("RemoteAddr", r.RemoteAddr),
		slog.String("clientID", cs.GetClientID()),
		slog.Int("nr sse connections", cs.GetNrConnections()),
	)
	//var sseMsg SSEEvent

	done := false
	for !done { // sseMsg := range sseChan {
		// wait for message, or writer closing
		select {
		case sseMsg, ok := <-sseChan: // received event

			if !ok { // channel was closed by session
				done = true
				break
			}
			slog.Info("SseHandler: sending sse event to client",
				slog.String("remote", r.RemoteAddr),
				slog.String("clientID", cs.GetClientID()),
				slog.String("eventType", sseMsg.EventType),
			)
			// WARNING: messages are send as MIME type "text/event-stream", which is defined as
			// "Each message is sent as a block of text terminated by a pair of newlines. "
			//https://developer.mozilla.org/en-US/docs/Web/API/Server-sent_events/Using_server-sent_events
			//_, err := fmt.Fprintf(w, "event: time\ndata: <div sse-swap='time'>%s</div>\n\n", data)
			// FIXME: There seems to be a 64K limit. Probably at the go-sse receiver side
			n, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n",
				sseMsg.EventType, sseMsg.Payload)
			_ = n
			//_, err := fmt.Fprint(w, sseMsg)
			if err != nil {
				slog.Error("Error writing event", "event", sseMsg.EventType,
					"size", len(sseMsg.Payload))
			}
			w.(http.Flusher).Flush()
			break
		case <-r.Context().Done(): // remote client connection closed
			slog.Info("Remote client disconnected (read context)")
			//close(sseChan)
			done = true
			break
		}
	}
	cs.DeleteSSEChan(sseChan)
	// TODO: if all connections are closed for this client send a disconnected event

	slog.Info("SseHandler: sse connection closed",
		slog.String("remote", r.RemoteAddr),
		slog.String("clientID", cs.GetClientID()),
		slog.Int("nr sse connections", cs.GetNrConnections()),
	)
}

// registerGosse using go-sse
func (svc *SSEHandler) registerWithGosse(r chi.Router) {

	svc.gosse.OnSession = func(sseSession *sse.Session) (sse.Subscription, bool) {
		// An active session is required before accepting the request. This is created on
		// authentication/login.
		cs, err := sessions.GetSessionFromContext(sseSession.Req)
		if err != nil {
			return sse.Subscription{}, false
		}
		slog.Info("SseHandler. New connection",
			slog.String("ClientID", cs.GetClientID()),
			slog.String("RemoteAddr", sseSession.Req.RemoteAddr))

		// TODO use subscription topics
		// TODO use last event ID
		//lastEventID := htSession.lastEventID
		sub := sse.Subscription{
			Client:      sseSession,
			LastEventID: sseSession.LastEventID,
			Topics:      []string{sse.DefaultTopic},
		}

		// establish a client event channel for sending messages back to the client
		ctx := sseSession.Req.Context()
		sseChan := cs.CreateSSEChan()
		done := false
		go func() {
			for !done {
				select {
				case ev, ok := <-sseChan:
					if !ok { // channel was closed by session
						done = true
						break
					}

					sseMsg := sse.Message{}
					sseMsg.AppendData(ev.Payload)
					sseMsg.ID, _ = sse.NewID(ev.ID)
					sseMsg.Type, err = sse.NewType(ev.EventType)
					err = sseSession.Send(&sseMsg)
					if err != nil {
						slog.Error("failed sending message", "err", err)
					} else {
						_ = sseSession.Flush()
					}

				case <-ctx.Done(): // remote client connection closed
					slog.Info("Remote client disconnected (req context)")
					done = true
					break
				}
			}
			cs.DeleteSSEChan(sseChan)
			// TODO: if all connections are closed for this client send a disconnected event
			slog.Info("SseHandler: Session closed", "clientID", cs.GetClientID())
		}()

		return sub, true
	}
	r.HandleFunc(vocab.ConnectSSEPath, svc.gosse.ServeHTTP)
}

// RegisterMethods registers handlers with the router
func (svc *SSEHandler) RegisterMethods(r chi.Router) {
	// built-in server or go-sse server
	//r.HandleFunc(vocab.ConnectSSEPath, svc.handleSseConnect)
	svc.registerWithGosse(r)
}

func (svc *SSEHandler) Stop() {
	svc.gosse.Shutdown(context.Background())
}

// NewSSEHandler creates an instance of the SSE connection handler
// handleMessage is used to send connect/disconnect events.
func NewSSEHandler(sessionAuth api.IAuthenticator) *SSEHandler {
	handler := SSEHandler{
		sessionAuth: sessionAuth,
		//handleMessage: handleMessage,
		gosse: &sse.Server{},
	}
	return &handler
}
