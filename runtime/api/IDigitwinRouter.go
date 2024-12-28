package api

import (
	"github.com/hiveot/hub/transports"
)

// IDigitwinRouter is the interface for routing the action,event and property messages
// received from consumers and agents. It handles the flow for TD level operations.
type IDigitwinRouter interface {

	// HandleRequest with the message to handle.
	// If a result is immediately available it is returned, otherwise it is sent
	// separately to the connection with the ID of replyTo
	HandleRequest(msg transports.RequestMessage, replyTo string) transports.ResponseMessage

	HandleResponse(resp transports.ResponseMessage) error

	HandleNotification(notif transports.NotificationMessage)
}
