package wssclient

// ActionStatusMessage containing progress of an action or property write request
type ActionStatusMessage struct {
	ThingID   string `json:"thingId"`
	Operation string `json:"messageType"` // WotOpPublishActionStatus
	Name      string `json:"action"`
	RequestID string `json:"requestID,omitempty"`
	// progress value: RequestDelivered, RequestCompleted, ...
	Progress string `json:"progress"`
	Error    string `json:"error"`
	// Output optional output data in case of a completed action
	Output any `json:"reply,omitempty"`
	// timestamp of the progress update
	Timestamp string `json:"timestamp"`
}

type EventMessage struct {
	ThingID   string `json:"thingId"`
	Operation string `json:"messageType"` // WotOpPublishEvent
	Name      string `json:"event"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestID,omitempty"`
	Timestamp string `json:"timestamp"`
}

type InvokeActionMessage struct {
	ThingID   string `json:"thingId"`
	Operation string `json:"messageType"` // WotOpInvokeAction
	Name      string `json:"action"`
	RequestID string `json:"requestID,omitempty"`
	Input     any    `json:"input,omitempty"`
	// Time the action was created
	Timestamp string `json:"timestamp"`
}

type ObservePropertyMessage struct {
	ThingID   string `json:"thingId"`
	Operation string `json:"messageType"` // WotOpObserveProperty,WotOpUnobserveProperty
	Name      string `json:"property"`
}

type PropertyMessage struct {
	ThingID   string `json:"thingId"`
	Operation string `json:"messageType"` // WotOpPublishProperty
	Name      string `json:"property"`
	Data      any    `json:"data,omitempty"`
	// optional request ID in case property update is the result of write property operation
	RequestID string `json:"requestID,omitempty"`
	// time the property value was read
	Timestamp string `json:"timestamp"`
}

type SubscribeMessage struct {
	ThingID   string `json:"thingId"`
	Operation string `json:"messageType"` // WotOpSubscribeEvent, WotOpUnsubscribeEvent
	Name      string `json:"event"`
}
