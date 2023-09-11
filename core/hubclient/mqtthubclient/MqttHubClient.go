package mqtthubclient

import (
	"github.com/eclipse/paho.golang/paho"
	"time"
)

const keepAliveInterval = 30 // seconds
const reconnectDelay = 10 * time.Second
const withDebug = false

// MqttHubClient manages the hub server connection with hub event and action messaging
// This implements the IHubClient interface.
// This implementation is based on the Mqtt messaging system.
type MqttHubClient struct {
	clientID string
	hostName string
	port     int
	pcl      *paho.Client
	timeout  time.Duration // request timeout
}

// NewMqttHubClient creates a new instance of the hub client using the connected paho client
func NewMqttHubClient(clientID string, pcl *paho.Client) *MqttHubClient {
	hc := &MqttHubClient{
		clientID: clientID,
		pcl:      pcl,
		timeout:  time.Second * 10,
	}
	return hc
}
