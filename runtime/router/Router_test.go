package router_test

import (
	"fmt"
	vocab "github.com/hiveot/hub/api/go"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/things"
	"github.com/hiveot/hub/runtime/router"
	"github.com/stretchr/testify/assert"
	"os"
	"strings"
	"testing"
)

func TestMain(m *testing.M) {
	logging.SetLogging("info", "")
	res := m.Run()
	os.Exit(res)
}

func TestHandleAction(t *testing.T) {
	serviceID := "service1"
	methodID := "method1"
	data := "Hello World"
	cfg := router.NewRouterConfig()
	r := router.NewMessageRouter(&cfg)

	// this handler returns the uppercase text
	r.AddServiceHandler(serviceID,
		func(tv *things.ThingMessage) ([]byte, error) {
			upper := strings.ToUpper(string(tv.Data))
			return []byte(upper), nil
		})
	tv1 := things.NewThingMessage(
		vocab.MessageTypeAction, serviceID, methodID, []byte(data), "sender1")
	resp, err := r.HandleMessage(tv1)
	assert.Equal(t, strings.ToUpper(string(tv1.Data)), string(resp))
	assert.NoError(t, err)
}

func TestHandleEvent(t *testing.T) {
	mwh1Count := 0
	mwh2Count := 0
	cfg := router.NewRouterConfig()
	r := router.NewMessageRouter(&cfg)

	r.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		mwh1Count++
		return tv, nil
	})
	r.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		mwh2Count++
		return tv, nil
	})
	r.AddServiceHandler("",
		func(tv *things.ThingMessage) ([]byte, error) {
			return tv.Data, nil
		})
	tv1 := things.NewThingMessage(vocab.MessageTypeEvent, "thing1", "key1", []byte("data"), "sender1")
	resp, err := r.HandleMessage(tv1)
	assert.Equal(t, tv1.Data, resp)
	assert.NoError(t, err)

	assert.Equal(t, mwh1Count, 1)
	assert.Equal(t, mwh2Count, 1)
}
func TestBadMessageType(t *testing.T) {
	cfg := router.NewRouterConfig()
	r := router.NewMessageRouter(&cfg)

	r.AddServiceHandler("",
		func(tv *things.ThingMessage) ([]byte, error) {
			return tv.Data, nil
		})
	tv1 := things.NewThingMessage("badmessagetype", "thing1", "key1", []byte("data"), "sender1")
	_, err := r.HandleMessage(tv1)
	assert.Error(t, err)
}

func TestMiddlewareError(t *testing.T) {
	cfg := router.NewRouterConfig()
	r := router.NewMessageRouter(&cfg)

	r.AddMiddlewareHandler(func(tv *things.ThingMessage) (*things.ThingMessage, error) {
		return tv, fmt.Errorf("middleware rejects message")
	})
	r.AddServiceHandler("",
		func(tv *things.ThingMessage) ([]byte, error) {
			return tv.Data, nil
		})
	tv1 := things.NewThingMessage(vocab.MessageTypeEvent, "thing1", "key1", []byte("data"), "sender1")
	_, err := r.HandleMessage(tv1)
	assert.Error(t, err)
}

func TestHandlerError(t *testing.T) {
	cfg := router.NewRouterConfig()
	r := router.NewMessageRouter(&cfg)

	r.AddServiceHandler("",
		func(tv *things.ThingMessage) ([]byte, error) {
			return tv.Data, fmt.Errorf("handler returns error")
		})
	tv1 := things.NewThingMessage(vocab.MessageTypeEvent, "thing1", "key1", []byte("data"), "sender1")
	_, err := r.HandleMessage(tv1)
	assert.Error(t, err)
}
