package transports

// ISubscription interface to underlying subscription mechanism
type ISubscription interface {
	Unsubscribe() error
}

// IHubTransport defines the interface of the message bus transport used by
// the hub client.
type IHubTransport interface {

	// AddressTokens returns the address separator and wildcard tokens used by the transport
	//  sep is the address separator. eg "." for nats, "/" for mqtt and redis
	//  wc is the address wildcard. "*" for nats, "+" for mqtt
	//  rem is the address remainder. "" for nats; "#" for mqtt
	AddressTokens() (sep, wc, rem string)

	// ConnectWithPassword connects to the messaging server using password authentication
	//  loginID is the client's ID
	//  password is created when registering the user with the auth service.
	ConnectWithPassword(password string) error

	// ConnectWithToken connects to the messaging server using an authentication token
	// and pub/private keys provided when creating an instance of the hub client.
	//  serializedKP is the key generated with CreateKey.
	//  token is created by the auth service.
	ConnectWithToken(serializedKP string, token string) error

	// CreateKeyPair returns a new set of serialized public/private key pair.
	CreateKeyPair() (serializedKeyPair string, serializedPub string)

	// Disconnect from the message bus
	Disconnect()

	// Pub for publications on any address
	Pub(address string, payload []byte) error

	// PubRequest publishes a RPC request and waits for a response
	PubRequest(address string, payload []byte) (resp []byte, err error)

	// Sub for subscribing to any address.
	Sub(address string, cb func(addr string, data []byte)) (ISubscription, error)

	// SubRequest subscribes to RPC requests and sends the reply to the sender
	SubRequest(address string, handler func(addr string, payload []byte) (
		reply []byte, err error)) (ISubscription, error)
}
