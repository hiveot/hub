// Package mqtthubclient.
// This is a copy of paho's rpc.go with a fix in Handler.Request
// See also: https://github.com/eclipse/paho.golang/pull/111/
//
// To prevent lockup if Request is not responded to, this adds the
// fix applied in autopaho.rpc

package mqtthubclient

import (
	"context"
	"fmt"
	"golang.org/x/exp/slog"
	"sync"
	"time"

	"github.com/eclipse/paho.golang/paho"
)

// InboxTopicFormat is the INBOX subscription topic used by the client
// Used to send replies to requests.
const InboxTopicFormat = "_INBOX.%s"

// Handler is the struct providing a request/response functionality for the paho
// MQTT v5 client
type Handler struct {
	sync.Mutex
	c          *paho.Client
	correlData map[string]chan *paho.Publish
}

func NewHandler(ctx context.Context, c *paho.Client) (*Handler, error) {
	h := &Handler{
		c:          c,
		correlData: make(map[string]chan *paho.Publish),
	}

	c.Router.RegisterHandler(fmt.Sprintf(InboxTopicFormat, c.ClientID), h.responseHandler)

	inboxTopic := fmt.Sprintf(InboxTopicFormat, c.ClientID)
	slog.Info("NewHandler. Subscribing to inbox", "topic", inboxTopic)
	_, err := c.Subscribe(ctx, &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			inboxTopic: {QoS: 1},
		},
	})
	if err != nil {
		return nil, err
	}

	return h, nil
}

func (h *Handler) addCorrelID(cID string, r chan *paho.Publish) {
	h.Lock()
	defer h.Unlock()

	h.correlData[cID] = r
}

func (h *Handler) getCorrelIDChan(cID string) chan *paho.Publish {
	h.Lock()
	defer h.Unlock()

	rChan := h.correlData[cID]
	delete(h.correlData, cID)

	return rChan
}

func (h *Handler) Request(ctx context.Context, pb *paho.Publish) (*paho.Publish, error) {
	cID := fmt.Sprintf("%d", time.Now().UnixNano())
	rChan := make(chan *paho.Publish)

	h.addCorrelID(cID, rChan)

	if pb.Properties == nil {
		pb.Properties = &paho.PublishProperties{}
	}

	pb.Properties.CorrelationData = []byte(cID)
	//HS 2023-09-10: allow override of response topic
	if pb.Properties.ResponseTopic == "" {
		pb.Properties.ResponseTopic = fmt.Sprintf(InboxTopicFormat, h.c.ClientID)
	}
	pb.Retain = false

	_, err := h.c.Publish(ctx, pb)
	if err != nil {
		return nil, err
	}
	// HS 2023-09-10: fix hangup when no response is received
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case resp := <-rChan:
		return resp, nil
	}
}

func (h *Handler) responseHandler(pb *paho.Publish) {
	if pb.Properties == nil || pb.Properties.CorrelationData == nil {
		return
	}

	rChan := h.getCorrelIDChan(string(pb.Properties.CorrelationData))
	if rChan == nil {
		return
	}

	rChan <- pb
}
