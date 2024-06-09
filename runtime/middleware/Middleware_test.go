package middleware_test

import (
	"fmt"
	"github.com/hiveot/hub/api/go/vocab"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/middleware"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

func TestHandleEvent(t *testing.T) {
	const payload = "hello"
	mwh1Count := 0
	mwh2Count := 0

	mw := middleware.NewMiddleware()
	mw.SetMessageHandler(func(msg *things.ThingMessage) hubclient.DeliveryStatus {
		var res hubclient.DeliveryStatus
		res.Status = hubclient.DeliveryCompleted
		res.Reply = msg.Data
		return res
	})

	mw.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		mwh1Count++
		return tv, nil
	})
	mw.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		mwh2Count++
		return tv, nil
	})

	tv1 := things.NewThingMessage(vocab.MessageTypeEvent, "thing1", "key1", []byte(payload), "sender1")
	stat := mw.HandleMessage(tv1)
	assert.Empty(t, stat.Error)
	assert.Equal(t, payload, string(stat.Reply))
	assert.Equal(t, mwh1Count, 1)
	assert.Equal(t, mwh2Count, 1)
}

func TestHandlerError(t *testing.T) {
	const payload = "hello"
	mw := middleware.NewMiddleware()
	mw.SetMessageHandler(func(msg *things.ThingMessage) hubclient.DeliveryStatus {
		var res hubclient.DeliveryStatus
		res.Status = hubclient.DeliveryFailed
		res.Error = "Failed reply"
		return res
	})
	mwh1Count := 0

	mw.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		mwh1Count++
		return tv, nil
	})

	tv1 := things.NewThingMessage(vocab.MessageTypeEvent, "thing1", "key1", []byte(payload), "sender1")
	stat := mw.HandleMessage(tv1)
	assert.Equal(t, mwh1Count, 1)
	assert.NotEmpty(t, stat.Error)
	assert.Equal(t, hubclient.DeliveryFailed, stat.Status)
}

func TestMiddlewareError(t *testing.T) {
	const payload = "hello"
	mwh1Count := 0
	mwh2Count := 0

	mw := middleware.NewMiddleware()
	mw.SetMessageHandler(func(msg *things.ThingMessage) hubclient.DeliveryStatus {
		assert.Fail(t, "should not get here")
		var res hubclient.DeliveryStatus
		return res
	})

	mw.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		mwh1Count++
		return tv, fmt.Errorf("this is a error for testing error handling")
	})

	mw.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		mwh2Count++
		return tv, nil
	})

	msg := things.NewThingMessage(vocab.MessageTypeEvent, "thing1", "key1", []byte(payload), "sender1")
	stat := mw.HandleMessage(msg)
	assert.NotEmpty(t, stat.Error)
	assert.Equal(t, hubclient.DeliveryFailed, stat.Status)
	assert.Equal(t, mwh1Count, 1)
	assert.Equal(t, mwh2Count, 0)
}
