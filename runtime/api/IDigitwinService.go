package api

// TODO: tdd2go should generate these

const DigitwinServiceID = "dtw:digitwin:service"

const ProgressEventName = "progress"

// Progress event sent on write property and invoke action
type ProgressEvent struct {
	ID          string `json:"ID"`
	Name        string `json:"name"`
	Data        any    `json:"data,omitempty"`
	MessageID   string `json:"messageID"`
	MessageType string `json:"messageType"`
	SenderID    string `json:"senderID"`
	Status      string `json:"status"`
	StatusInfo  string `json:"statusInfo,omitempty"`
}
