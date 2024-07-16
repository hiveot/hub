package sseserver

import (
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/runtime/transports/httpstransport/sessions"
	"github.com/tmaxmax/go-sse"
	"log/slog"
)

func NewGoSSEServer() *sse.Server {

	gosse := &sse.Server{}
	gosse.OnSession = func(sseSession *sse.Session) (sse.Subscription, bool) {
		// An active session is required before accepting the request. This is created on
		// authentication/login.
		cs, err := sessions.GetSessionFromContext(sseSession.Req)
		if err != nil {
			return sse.Subscription{}, false
		}
		slog.Warn("SseHandler. New SSE connection",
			slog.String("ClientID", cs.GetClientID()),
			slog.String("RemoteAddr", sseSession.Req.RemoteAddr),
			slog.String("protocol", sseSession.Req.Proto),
		)

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
		sseChan := cs.CreateSSEChan() // for this server
		// Send a ping event as the go-sse client doesn't have a 'connected callback'
		sseChan <- sessions.SSEEvent{EventType: hubclient.PingMessage}

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
					// FIXME: it causes a race condition in http
					//err = sseSession.Send(&sseMsg)
					err = sub.Client.Send(&sseMsg)
					if err != nil {
						slog.Error("failed sending message", "err", err)
					} else {
						//_ = sseSession.Flush()
						_ = sub.Client.Flush()
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

	return gosse
}
