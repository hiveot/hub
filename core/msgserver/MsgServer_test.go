package msgserver_test

import (
	"fmt"
	"github.com/hiveot/hub/core/auth/authapi"
	"github.com/hiveot/hub/lib/logging"
	"github.com/hiveot/hub/lib/testenv"
	"github.com/hiveot/hub/lib/things"
	"sync/atomic"
	"testing"
	"time"
)

const core = "nats"

// Benchmark a simple event pub/sub
// mqtt: 95 usec/rpc (QOS 1), 60 usec/rpc (QOS 0); 450 usec for request/response
// nats:  9 usec/rpc ! (* risk of dropped messages); 170 usec for request/response
func Benchmark_PubSubEvent(b *testing.B) {
	logging.SetLogging("warning", "")
	txCount := atomic.Int32{}
	rxCount := atomic.Int32{}

	ts, _ := testenv.StartTestServer(core, false)
	defer ts.Stop()

	cl1, _ := ts.AddConnectClient("publisher", authapi.ClientTypeDevice, authapi.ClientRoleDevice)
	defer cl1.Disconnect()
	cl2, _ := ts.AddConnectClient("sub", authapi.ClientTypeUser, authapi.ClientRoleOperator)
	defer cl2.Disconnect()
	_ = cl2.SubEvents("publisher", "", "")
	cl2.SetEventHandler(func(msg *things.ThingValue) {
		//time.Sleep(time.Millisecond * 10)
		rxCount.Add(1)
	})

	t1 := time.Now()
	b.Run("pub and sub event",
		func(b *testing.B) {
			for n := 0; n < b.N; n++ {
				txCount.Add(1)
				// nats loses events without a minor delay
				// FIXME: a small delay to prevent events from being dropped when using NATS
				// docs (https://docs.nats.io/nats-concepts/what-is-nats) say to use jetstream for guaranteed delivery.
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

	cl1, _ := ts.AddConnectClient("client1", authapi.ClientTypeUser, authapi.ClientRoleAdmin)
	defer cl1.Disconnect()
	cl2, _ := ts.AddConnectClient("rpc", authapi.ClientTypeService, authapi.ClientRoleService)
	defer cl2.Disconnect()

	cl2.SetRPCHandler(func(msg *things.ThingValue) ([]byte, error) {
		rxCount.Add(1)
		return msg.Data, nil
	})

	t1 := time.Now()
	b.Run("pub and sub request",
		func(b *testing.B) {

			for n := 0; n < b.N; n++ {
				txCount.Add(1)
				req := "request"
				repl := ""
				err := cl1.PubRPCRequest("rpc", "cap1", "method1", &req, &repl)
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
