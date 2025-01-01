package base

import (
	"context"
	"github.com/hiveot/hub/transports"
	"log/slog"
	"sync"
	"time"
)

// RnRChan is a helper for request/response message handling using channels.
// Intended to link responses in asynchronous request-response communication.
//
// Usage:
//  1. create a request ID: shortid.MustGenerate()
//  2. register the request ID: c := Open(requestID)
//  3. Send the request message in the client, passing the requestID
//  4. Wait for a response: completed, data := WaitForResponse(c, timeout)
//  5. Handle response message (in client callback): HandleResponse(requestID,data)
type RnRChan struct {
	mux sync.RWMutex

	// map of requestID to delivery status update channel
	correlData map[string]chan transports.ResponseMessage

	//timeout write to a response channel
	writeTimeout time.Duration
}

// Close removes the request channel
func (rnr *RnRChan) Close(requestID string) {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()
	rChan, found := rnr.correlData[requestID]
	if found {
		delete(rnr.correlData, requestID)
		close(rChan)
	}
}

// CloseAll force closes all channels, ending all waits for RPC responses.
func (rnr *RnRChan) CloseAll() {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()
	for _, rChan := range rnr.correlData {
		close(rChan)
	}
	rnr.correlData = make(map[string]chan transports.ResponseMessage)

}

// HandleResponse writes a reply to the request channel.
//
// This returns true on success or false if requestID is unknown (no-one is waiting)
//
// If a timeout passes while writing is block the write is released.
func (rnr *RnRChan) HandleResponse(msg transports.ResponseMessage) bool {
	rnr.mux.Lock()
	rChan, isRPC := rnr.correlData[msg.RequestID]
	rnr.mux.Unlock()
	if isRPC {
		ctx, cancelFn := context.WithTimeout(context.Background(), rnr.writeTimeout)
		select {
		case rChan <- msg:
		case <-ctx.Done():
			slog.Error("Response channel is full. Is no-one listening?")
			// pass as notification as the channel isn't read
			isRPC = false
		}
		cancelFn()
	}
	return isRPC
}

func (rnr *RnRChan) Len() int {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()
	return len(rnr.correlData)
}

// Open a new channel for receiving response to a request
// Call Close when done.
//
// This returns a reply channel on which the data is received. Use
// WaitForResponse(rChan)
func (rnr *RnRChan) Open(requestID string) chan transports.ResponseMessage {
	// todo: is there a use-case for a buffer?
	rChan := make(chan transports.ResponseMessage)
	rnr.mux.Lock()
	rnr.correlData[requestID] = rChan
	rnr.mux.Unlock()
	return rChan
}

// WaitForResponse waits for an answer received on the reply channel.
// After timeout without response this returns with completed is false.
//
// Use 'autoclose' to immediately close the channel when no further responses are
// expected. (they would not be read and thus lost)
//
// If the channel was closed this returns completed with no reply
func (rnr *RnRChan) WaitForResponse(
	replyChan chan transports.ResponseMessage, timeout time.Duration) (
	completed bool, resp transports.ResponseMessage) {

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	select {
	case rData := <-replyChan:
		resp = rData
		completed = true
		break
	case <-ctx.Done():
		completed = false
	}
	return completed, resp
}

func NewRnRChan() *RnRChan {
	r := &RnRChan{
		correlData:   make(map[string]chan transports.ResponseMessage),
		writeTimeout: time.Second * 300, // default 3
	}
	return r
}
