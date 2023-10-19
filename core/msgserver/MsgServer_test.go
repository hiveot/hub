package msgserver_test

import (
	"fmt"
	"github.com/hiveot/hub/core/auth"
	"github.com/hiveot/hub/lib/hubclient"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/thing"
	"sync/atomic"
	"testing"
	"time"
)

const core = "mqtt"

// Benchmark a simple event pub/sub
// mqtt: 100 usec/rpc   (with ack?)
// nats:   8 usec/rpc ! (* risk of dropped messages)
func Benchmark_PubSubEvent(b *testing.B) {
	logging.SetLogging("warning", "")
	txCount := atomic.Int32{}
	rxCount := atomic.Int32{}

	ts, _ := testenv.StartTestServer(core, false)
	defer ts.Stop()

	cl1, _ := ts.AddConnectClient("publisher", auth.ClientTypeDevice, auth.ClientRoleDevice)
	defer cl1.Disconnect()
	cl2, _ := ts.AddConnectClient("sub", auth.ClientTypeUser, auth.ClientRoleOperator)
	defer cl2.Disconnect()

	sub2, _ := cl2.SubEvents("publisher", "", "", func(msg *thing.ThingValue) {
		//time.Sleep(time.Millisecond * 10)
		rxCount.Add(1)
	})
	defer sub2.Unsubscribe()

	t1 := time.Now()
	b.Run("pub and sub",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				txCount.Add(1)
				// nats loses events without a minor delay
				// FIXME: check nats delivery options for guaranteed once
				time.Sleep(time.Microsecond)
				_ = cl1.PubEvent("thing1", "ev1", []byte("hello"))
			}
		})

	d1 := time.Now().Sub(t1)
	fmt.Printf("==> Duration: %d usec per rq\n\n", d1.Microseconds()/int64(txCount.Load()))

	time.Sleep(time.Millisecond * 10)
	if rxCount.Load() != txCount.Load() {
		b.Error(fmt.Sprintf("rx count %d doesn't match tx count %d", rxCount.Load(), txCount.Load()))
	}
}

// benchmark RPC request
// mqtt: 300 usec/request
// nats: 120 usec/request
func Benchmark_Request(b *testing.B) {
	logging.SetLogging("warning", "")
	txCount := atomic.Int32{}
	rxCount := atomic.Int32{}

	ts, _ := testenv.StartTestServer(core, false)
	defer ts.Stop()

	cl1, _ := ts.AddConnectClient("client1", auth.ClientTypeUser, auth.ClientRoleAdmin)
	defer cl1.Disconnect()
	cl2, _ := ts.AddConnectClient("rpc", auth.ClientTypeService, auth.ClientRoleService)
	defer cl2.Disconnect()

	sub2, _ := cl2.SubRPCRequest("cap1", func(msg *hubclient.RequestMessage) error {
		rxCount.Add(1)
		//time.Sleep(time.Millisecond * 10)
		_ = msg.SendReply(msg.Payload, nil)
		return nil
	})
	defer sub2.Unsubscribe()

	t1 := time.Now()
	b.Run("pub and sub",
		func(b *testing.B) {

			for n := 0; n < b.N; n++ {
				txCount.Add(1)
				req := "request"
				repl := ""
				ar, err := cl1.PubRPCRequest("rpc", "cap1", "method1", &req, &repl)
				_ = ar
				_ = err
				if req != repl {
					b.Error("request doesn't match reply")
				}
			}
		})
	d1 := time.Now().Sub(t1)
	fmt.Printf("==> Duration: %d usec per rq\n\n", d1.Microseconds()/int64(txCount.Load()))
	time.Sleep(time.Millisecond)
	if rxCount.Load() != txCount.Load() {
		b.Error(fmt.Sprintf("rx count %d doesn't match tx count %d", rxCount.Load(), txCount.Load()))
	}
}
