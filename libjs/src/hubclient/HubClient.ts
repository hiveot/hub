import { EventTypes, MessageType } from '../vocab/vocabulary.js';
import type { ThingTD } from '../things/ThingTD.js';
import type { IHubTransport } from "./transports/IHubTransport.js";
import { ConnInfo, ConnectionStatus } from "./transports/IHubTransport.js";
import type { ThingValue } from "../things/ThingValue";
import { MqttTransport } from "./transports/mqtttransport/MqttTransport.js";
import type { IHiveKey } from "@hivelib/keys/IHiveKey";
import { NatsTransport } from './transports/natstransport/NatsTransport.js';
import * as tslog from 'tslog';


const log = new tslog.Logger()

// HubClient implements the javascript client for connecting to the hub,
// using one of available transports.
export class HubClient {
	tp: IHubTransport;
	_clientID: string;
	isInitialized: boolean = false;
	connStatus: ConnectionStatus;
	connInfo: string;

	// client handler for action requests of things published by this client, if any.
	actionHandler: ((tv: ThingValue) => string) | null = null;
	// client handler for config changes to things published by this client, if any.
	configHandler: ((tv: ThingValue) => boolean) | null = null;
	// client handler for connection stastus change
	connectHandler: ((status: ConnectionStatus, info: ConnInfo) => void) | null = null
	// client handler for subscribed events
	eventHandler: ((tv: ThingValue) => void) | null = null;
	// client handler for rpc requests of capabilities of this client, if any.
	rpcHandler: ((tv: ThingValue) => string) | null = null;

	// Hub Client
	// @param transport is a connected transport
	// @param clientID connected as
	constructor(tp: IHubTransport, clientID: string) {
		this._clientID = clientID;
		this.tp = tp;
		this.connStatus = ConnectionStatus.Disconnected;
		this.connInfo = ConnInfo.NotConnected;

		tp.setConnectHandler(this.onConnection.bind(this));
	}

	// MakeAddress creates a message address optionally with wildcards
	// This uses the hiveot address format: {msgType}/{deviceID}/{thingID}/{name}[/{clientID}]
	// Where '/' is the address separator for MQTT or '.' for Nats
	// Where "+" is the wildcard for MQTT or "*" for Nats
	//
	//	msgType is the message type: "event", "action", "config" or "rpc".
	//	agentID is the device or service being addressed. Use "" for wildcard
	//	thingID is the ID of the thing managed by the publisher. Use "" for wildcard
	//	name is the event or action name. Use "" for wildcard.
	//	clientID is the login ID of the sender. Use "" for subscribe.
	_makeAddress(msgType: string, agentID: string, thingID: string,
		name: string, clientID: string): string {

		let { sep, wc, rem } = this.tp.addressTokens()
		let addr: string

		if (msgType == "") {
			msgType = MessageType.Event
		}
		if (agentID == "") {
			agentID = wc
		}
		if (thingID == "") {
			thingID = wc
		}
		if (name == "") {
			name = wc
		}
		if (clientID == "") {
			clientID = rem
		}
		addr = msgType + sep + agentID + sep + thingID + sep + name + sep + clientID
		return addr
	}


	// SplitAddress separates an address into its components
	//
	// addr is a hiveot address eg: msgType/things/deviceID/thingID/name/clientID
	_splitAddress(addr: string): {
		msgType: string, agentID: string, thingID: string,
		name: string, senderID: string, err: Error | null
	} {

		let msgType: string = "",
			agentID: string = "",
			thingID: string = "",
			name: string = "",
			senderID: string = "",
			err: Error | null = null

		let { sep, wc, rem } = this.tp.addressTokens();

		let parts = addr.split(sep)

		// inbox topics are short
		if (parts.length >= 1 && parts[0] == MessageType.INBOX) {
			msgType = parts[0]
			if (parts.length >= 2) {
				agentID = parts[1]
			}
		} else if (parts.length < 4) {
			err = new Error("incomplete address")
		} else if (parts.length == 4) {
			msgType = parts[0]
			agentID = parts[1]
			thingID = parts[2]
			name = parts[3]
		} else if (parts.length > 4) {
			msgType = parts[0]
			agentID = parts[1]
			thingID = parts[2]
			name = parts[3]
			senderID = parts[4]
		}
		return { msgType, agentID, thingID, name, senderID, err }
	}

	// ClientID the client is authenticated as to the server
	get clientID(): string {
		return this._clientID;
	}

	// return the current connection status
	get connectionStatus(): { status: ConnectionStatus, info: string } {
		return { status: this.connStatus, info: this.connInfo };
	}

	// connect and login to the Hub gateway using a JWT token
	// host is the server address
	async connectWithToken(kp: IHiveKey | undefined, jwtToken: string) {

		// pass-through to the underlying transport
		if (kp) {
			return this.tp.connectWithToken(kp, jwtToken);
		}
		return
	}

	// ConnectWithPassword connects to the Hub server using the clientID and password.
	async connectWithPassword(password: string) {
		// pass-through to the underlying transport
		await this.tp.connectWithPassword(password);

		// receive actions and config requests for this agent.
		let addr = this._makeAddress(MessageType.Action, this.clientID, "", "", "")
		this.tp.subscribe(addr)
		addr = this._makeAddress(MessageType.Config, this.clientID, "", "", "")
		this.tp.subscribe(addr)
	}


	createKeyPair(): IHiveKey {
		let kp = this.tp.createKeyPair()
		return kp
	}
	// disconnect if connected
	async disconnect() {
		if (this.connStatus != ConnectionStatus.Disconnected) {
			this.tp.disconnect()
		}
	}

	// callback handler invoked when the connection status has changed
	onConnection(status: ConnectionStatus, info: string) {
		this.connStatus = status
		this.connInfo = info
		if (this.connStatus == ConnectionStatus.Connected) {
			log.info('HubClient connected');
		} else if (this.connStatus == ConnectionStatus.Connecting) {
			log.info('HubClient attempt connecting');
		} else {
			log.info('HubClient disconnected');
		}
	}


	// Handle incoming messages and pass them to the event handler
	onEvent(addr: string, payload: string): void {
		let { msgType, agentID, thingID, name, senderID, err } =
			this._splitAddress(addr)
		let timestampMsec = Date.now() // UTC in msec
		let tv: ThingValue = {
			agentID: agentID,
			thingID: thingID,
			name: name,
			senderID: senderID,
			valueType: msgType,
			createdMSec: timestampMsec,
			data: payload,
		}
		if (err != null) {
			err = new Error("handleEvent: Received event on invalid address '" + addr + "': " + err.message)
			log.info(err)
			throw err
		}
		if (this.eventHandler != null) {
			this.eventHandler(tv)
		}
	}

	// Handle incoming action or config request messages and pass them on to their
	// respective handlers, if set.
	// The response will be sent back to the caller.
	onRequest(addr: string, payload: string): string {

		let { msgType, agentID, thingID, name, senderID, err } =
			this._splitAddress(addr)
		let timestampMsec = Date.now() // UTC in msec

		let tv: ThingValue = {
			agentID: agentID,
			thingID: thingID,
			name: name,
			senderID: senderID,
			valueType: msgType,
			createdMSec: timestampMsec,
			data: payload,
		}
		if (senderID == "") {
			err = new Error("handleRequest: Missing senderID on address '" + addr + ", request ignored.")
			log.info(err)
			throw err
		} else if (err != null) {
			err = new Error("handleRequest: Received request on invalid address '" + addr + "': " + err.message)
			log.info(err)
			throw err
		}
		// detection of request must have been mistaken as only subscriptions with this
		// clientID are made. 
		if (agentID != this._clientID) {
			err = new Error("request received for another agent");
			log.error(err)
			throw err
		}

		if (msgType == MessageType.Action && this.actionHandler != null) {
			return this.actionHandler(tv)
		} else if (msgType == MessageType.Config && this.configHandler != null) {
			let success = this.configHandler(tv)
			if (!success) {
				err = new Error("handleRequest: Config request not accepted")
				log.info(err)
				throw err
			} else {
				return ""
			}
		} else {
			err = Error("handleRequest: No handler is set for " + msgType + " messages")
			throw err
		}
	}


	// PubAction publishes a request for action from a Thing.
	//
	//	@param agentID: of the device or service that handles the action.
	//	@param thingID: is the destination thingID to whom the action applies.
	//	name is the name of the action as described in the Thing's TD
	//	payload is the optional serialized message of the action as described in the Thing's TD
	//
	// This returns the serialized reply data or null in case of no reply data
	async pubAction(agentID: string, thingID: string, name: string, payload: string): Promise<string | null> {
		log.info("pubAction. agentID:", agentID, ", thingID:", thingID, ", actionName:", name)
		let addr = this._makeAddress(MessageType.Action, agentID, thingID, name, this.clientID);
		let reply = await this.tp.pubRequest(addr, payload);
		if (typeof (reply) == "boolean") {
			return String(reply)
		}
		return reply
	}

	// PubAction publishes a request for changing a Thing's configuration.
	// The configuration is a writable property as defined in the Thing's TD.
	async pubConfig(agentID: string, thingID: string, propName: string, propValue: string): Promise<boolean> {
		log.info("pubConfig. agentID:", agentID, ", thingID:", thingID, ", propName:", propName)
		let addr = this._makeAddress(MessageType.Config, agentID, thingID, propName, this.clientID);
		let accepted = await this.tp.pubRequest(addr, propValue)
		return (!!accepted)
	}

	// PubEvent publishes a Thing event. The payload is an event value as per TD document.
	// Intended for devices and services to notify of changes to the Things they are the agent for.
	//
	// thingID is the ID of the 'thing' whose event to publish.
	// This is the ID under which the TD document is published that describes
	// the thing. It can be the ID of the sensor, actuator or service.
	//
	// This will use the client's ID as the agentID of the event.
	// eventName is the ID of the event described in the TD document 'events' section,
	// or one of the predefined events listed above as EventIDXyz
	//
	//	@param thingID: of the Thing whose event is published
	//	@param eventName: is one of the predefined events as described in the Thing TD
	//	@param payload: is the serialized event value, or nil if the event has no value
	async pubEvent(thingID: string, eventName: string, payload: string) {
		let addr = this._makeAddress(MessageType.Event, this.clientID, thingID, eventName, this.clientID)

		return this.tp.pubEvent(addr, payload)
	}

	// Publish a Thing property map
	// Ignored if props map is empty
	async pubProperties(thingID: string, props: { [key: string]: any }) {
		// if (length(props.) > 0) {
		let propsJSON = JSON.stringify(props, null, ' ');
		if (propsJSON.length > 2) {
			await this.pubEvent(thingID, EventTypes.Properties, propsJSON);
		}
	}

	// PubRPCRequest publishes an RPC request to a service and waits for a response.
	// Intended for users and services to invoke RPC to services.
	//
	// Authorization to use the service capability can depend on the user's role. Check the service
	// documentation for details. When unauthorized then an error will be returned after a short delay.
	//
	// The client's ID is used as the senderID of the rpc request.
	//
	//	agentID of the service that handles the request
	//	capability is the capability to invoke
	//	methodName is the name of the request method to invoke
	//	req is the request message that will be marshalled or nil if no arguments are expected
	//	returns a response message that is unmarshalled, or nil if no response is expected
	async pubRPCRequest(agentID: string, capability: string, methodName: string, req: any): Promise<any> {
		let addr = this._makeAddress(MessageType.RPC, agentID, capability, methodName, this.clientID);
		let payload = JSON.stringify(req, null, ' ')
		let reply = await this.tp.pubRequest(addr, payload);
		if (reply == "") {
			return ""
		} else if (reply == true || reply == false) {
			return reply
		}
		return JSON.parse(reply);
	}

	// PubTD publishes an event with a Thing TD document.
	// The client's authentication ID will be used as the agentID of the event.
	async pubTD(td: ThingTD) {
		let tdJSON = JSON.stringify(td, null, ' ');
		return this.pubEvent(td.id, EventTypes.TD, tdJSON);
	}


	// set the handler of thing action requests and subscribe to action requests
	setActionHandler(handler: (tv: ThingValue) => string) {
		this.actionHandler = handler
		// handler of requests is the same for actions, config and rpc
		this.tp.setRequestHandler(this.onRequest.bind(this))
		let addr = this._makeAddress(MessageType.Action, this.clientID, "", "", "")
		this.tp.subscribe(addr)
	}
	// set the handler of thing configuration requests
	setConfigHandler(handler: (tv: ThingValue) => boolean) {
		this.configHandler = handler
		// handler of requests is the same for actions, config and rpc
		this.tp.setRequestHandler(this.onRequest.bind(this))
		let addr = this._makeAddress(MessageType.Config, this.clientID, "", "", "")
		this.tp.subscribe(addr)
	}
	// set the handler of thing configuration requests
	setConnectionHandler(handler: (status: ConnectionStatus, info: string) => boolean) {
		this.connectHandler = handler
	}
	// set the handler for subscribed events
	setEventHandler(handler: (tv: ThingValue) => void): void {
		this.eventHandler = handler
		this.tp.setEventHandler(this.onEvent.bind(this))
	}
	// set the handler of rpc requests
	setRPCHandler(handler: (tv: ThingValue) => string) {
		this.rpcHandler = handler
		// handler of requests is the same for actions, config and rpc
		this.tp.setRequestHandler(this.onRequest.bind(this))
		let addr = this._makeAddress(MessageType.RPC, this.clientID, "", "", "")
		this.tp.subscribe(addr)

	}


	// Read Thing definitions from the directory
	// @param publisherID whose things to read or "" for all publishers
	// @param thingID whose to read or "" for all things of the publisher(s)
	// async readDirectory(agentID: string, thingID: string): Promise<string> {
	// 	return global.hapiReadDirectory(publisherID, thingID);
	// }

	// Subscribe to events from things. 
	//
	// The events will be passed to the configured onEvent handler.
	//
	// note there is no unsubscribe. The intended use is to subscribe to devices/things/events
	// of interest and leave it at that. Currently there is no use-case that requires
	// a frequent subscribe/unsubscribe.
	//
	// @param agentID: optional filter on the agent that publishes events; "" for all agents.
	// @param thingID: optional filter of the thing whose events are published; "" for all things
	// @param eventID: optional filter on the event name; "" for all event names.
	async subscribe(agentID: string, thingID: string, eventID: string): Promise<void> {

		let addr = this._makeAddress(MessageType.Event, agentID, thingID, eventID, "");

		return this.tp.subscribe(addr)
	}
}

// NewHubClientFromTransport returns a new Hub Client instance for the given transport.
//
//   - message bus transport to use, eg NatsTransport or MqttTransport instance
//   - clientID of the client that will be connecting
export function NewHubClientFromTransport(transport: IHubTransport, clientID: string): HubClient {
	let hc = new HubClient(transport, clientID)
	return hc
}


// NewHubClient returns a new Hub Client instance
//
// The keyPair string is optional. If not provided a new set of keys will be created.
// Use GetKeyPair to retrieve it for saving to file.
//
// Invoke hubClient.ConnectWithXy() to connect
//
//   - url of server to connect to.
//   - clientID of the client that will be connecting
//   - keyPair is this client's serialized private/public key pair, or "" to create them.
//   - caCertPem of server or "" to not verify server cert
//   - core server to use, "nats" or "mqtt". Default "" will use nats if url starts with "nats" or mqtt otherwise.
export function NewHubClient(url: string, clientID: string, caCertPem: string, core?: string): HubClient {
	let tp: IHubTransport
	if (core == "nats" || url.startsWith("nats")) {
		tp = new NatsTransport(url, clientID, caCertPem)
	} else {
		tp = new MqttTransport(url, clientID, caCertPem)
	}

	let hc = NewHubClientFromTransport(tp, clientID)

	return hc
}


