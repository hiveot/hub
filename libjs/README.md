# hivelibjs

Typescript library for connecting devices, services or web browser clients to the HiveOT Hub.

## Status

This is in development.

TODO:
* add MQTT websocket support for both node and browser
* add NATS transport
* add RPC clients for core services
	* auth
	* state
	* directory
	* history
	* idprov

* MQTT transport auth using signed nonce

* logging - something better than console
	* structured logging
	* formatting
	* timestamps

* testing
	* test things
	* test vocab
	* test on both browser and node
	* add test framework. jest is an option but it doesn't support tsx

## Modules

* discovery: Discovery of the Hub using DNS-SD
* hubclient: Client for publishing and subscribing of actions, events, td, and config
	* ./transports/mqtttransport: transport for use with MQTT message bus
	* ./transports/natstransport: transport for use with NATS message bus
* keys: keys for authentication and signing, using ED25519 or nkeys.
* things: building of W3C WoT TD documents
* vocab: HiveOT vocabulary used to describe properties, events and actions 

