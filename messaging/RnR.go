package messaging

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// RnRChan is a helper for Request 'n Response message handling using channels.
// Intended to link responses in asynchronous request-response communication.
//
// Usage:
//  1. create a request ID: shortid.MustGenerate()
//  2. register the request ID: c := Open(correlationID)
//  3. Send the request message in the client, passing the correlationID
//  4. Wait for a response: completed, data := WaitForResponse(c, timeout)
//  5. Handle response message (in client callback): HandleResponse(correlationID,data)
type RnRChan struct {
	mux sync.RWMutex

	// map of correlationID to delivery status update channel
	correlData map[string]chan *ResponseMessage

	//timeout write to a response channel
	writeTimeout time.Duration
}

// Close removes the request channel
func (rnr *RnRChan) Close(correlationID string) {
	rnr.mux.Lock()
	defer rnr.mux.Unlock()

	//slog.Info("closing channel. ", "correlationID", correlationID)
	rChan, found := rnr.correlData[correlationID]
	if found {
		delete(rnr.correlData, correlationID)
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
	rnr.correlData = make(map[string]chan *ResponseMessage)

}

// HandleResponse writes a reply to the request channel.
//
// This returns true on success or false if correlationID is unknown (no-one is waiting)
// It is up to the handler of this response to close the channel when done.
//
// If a timeout passes while writing is block the write is released.
func (rnr *RnRChan) HandleResponse(msg *ResponseMessage) bool {
	// Note: avoid a race between closing the channel and writing multiple responses.
	// This would happen if a 'pending' response arrives after a 'completed' response,
	// and 'wait-for-response' closes the channel while the second result is written.
	// This would panic, so lock the lookup and writing of the response channel.
	rnr.mux.Lock()
	rChan, isRPC := rnr.correlData[msg.CorrelationID]
	defer rnr.mux.Unlock()
	if isRPC {
		slog.Debug("HandleResponse: writing response to RPC go channel. ",
			slog.String("correlationID", msg.CorrelationID),
			slog.String("operation", msg.Operation),
		)
		ctx, cancelFn := context.WithTimeout(context.Background(), rnr.writeTimeout)
		select {
		case rChan <- msg:
		case <-ctx.Done():
			// this should never happen
			slog.Error("Response RPC go channel is full. Is no-one listening?")
			// recover
			isRPC = false
		}
		cancelFn()
	} else {
		//slog.Info("HandleResponse: not an RPC call (subscription).",
		//	slog.String("correlationID", msg.CorrelationID),
		//	slog.String("operation", msg.Operation))
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
func (rnr *RnRChan) Open(correlationID string) chan *ResponseMessage {
	//slog.Info("opening channel. ", "correlationID", correlationID)
	// this needs to be able to buffer 1 response in case completed and pending
	// are received out of order.
	rChan := make(chan *ResponseMessage, 1)
	rnr.mux.Lock()
	rnr.correlData[correlationID] = rChan
	rnr.mux.Unlock()
	return rChan
}

// WaitForResponse waits for an answer received on the reply channel.
// After timeout without response this returns with completed is false.
//
// Use 'autoclose' to immediately close the channel when no further responses are
// expected. (they would not be read and thus lost)
//
// If the channel was closed this returns hasResponse with no reply
func (rnr *RnRChan) WaitForResponse(
	replyChan chan *ResponseMessage, timeout time.Duration) (
	hasResponse bool, resp *ResponseMessage) {

	ctx, cancelFunc := context.WithTimeout(context.Background(), timeout)
	defer cancelFunc()
	select {
	case rData := <-replyChan:
		resp = rData
		hasResponse = true
		break
	case <-ctx.Done():
		hasResponse = false
	}
	return hasResponse, resp
}

func NewRnRChan() *RnRChan {
	r := &RnRChan{
		correlData:   make(map[string]chan *ResponseMessage),
		writeTimeout: time.Second * 300, // default 3
	}
	return r
}
