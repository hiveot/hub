package transports

import "github.com/hiveot/hub/lib/keys"

// ISubscription interface to underlying subscription mechanism
type ISubscription interface {
	Unsubscribe() error
}

// IHubTransport defines the interface of the message bus transport used by
// the hub client.
type IHubTransport interface {

	// AddressTokens returns the address separator and wildcard tokens used by the transport.
	//  sep is the address separator. eg "." for nats, "/" for mqtt and redis
	//  wc is the address wildcard. "*" for nats, "+" for mqtt
	//  rem is the address remainder. "" for nats; "#" for mqtt
	AddressTokens() (sep, wc, rem string)

	// ConnectWithPassword connects to the messaging server using password authentication.
	//  loginID is the client's ID
	//  password is created when registering the user with the auth service.
	ConnectWithPassword(password string) error

	// ConnectWithToken connects to the messaging server using an authentication token
	// and pub/private keys provided when creating an instance of the hub client.
	//  kp is the client's key pair
	//  token is created by the auth service.
	ConnectWithToken(kp keys.IHiveKey, token string) error

	// CreateKeyPair returns a new set of serialized public/private key pair.
	//  serializedKP contains the serialized public/private key pair
	//  pubKey contains the serialized public key to be shared
	CreateKeyPair() (kp keys.IHiveKey)

	// Disconnect from the message bus.
	Disconnect()

	// Pub for publications on any address.
	//	address to publish on
	//	payload with serialized message to publish
	Pub(address string, payload []byte) error

	// PubRequest publishes a RPC request and waits for a response.
	//  address to publish on
	//  payload with serialized message to publish
	//  returns a reply with serialized response message
	PubRequest(address string, payload []byte) (reply []byte, err error)

	// Sub for subscribing to an address.
	//	address to subscribe to, this can contain wildcards
	//	cb callback to invoke when a message is received
	Sub(address string, cb func(addr string, data []byte)) (ISubscription, error)

	// SubRequest subscribes to RPC requests and sends the reply to the sender.
	// Intended for services.
	//  address is the address to subscribe to (using AddressTokens to construct)
	//  cb is the callback to invoke when a message is received
	//
	// Returns a subscription object that needs to be unsubscribed when done
	SubRequest(address string, handler func(addr string, payload []byte) (
		reply []byte, err error)) (ISubscription, error)
}
