// Client to the hub gatewayclient
import ws from 'ws';
import type { IConnectionStatus } from './IConnectionStatus.js';
import * as lw from './startWasm.js';
import { EventTypes } from '$lib/hapi/vocabulary.js';
import type { ThingTD } from '$lib/hapi/thing.js';
import path from 'path';

// @ts-ignore   // global.d.ts doesn't seem to be used...???
globalThis.WebSocket = ws;

export class HubAPI {
	host: string = 'localhost';
	port: number = 9884;
	conStat: IConnectionStatus;
	isInitialized: boolean = false;

	// Gateway client. Start with blanks
	constructor(clientID: string) {
		this.conStat = {
			clientID: clientID,
			status: 'disconnected',
			statusMessage: 'Please login to continue'
		};
	}

	// Initialize the Hub API
	// This loads the Hub Client WASM file and waits until it is ready for use
	async initialize() {
		if (!this.isInitialized) {
			const wasmPath = path.join('build', 'hapi.wasm');
			console.log('initializing hapi from: ', wasmPath);
			await lw.startWasmServer(wasmPath);
			console.log('hapi initialized');
			this.isInitialized = true;
		} else {
			console.log('hapi already initialized');
		}
	}

	// return the current connection status
	get connectionStatus(): IConnectionStatus {
		return this.conStat;
	}

	// connect and login to the Hub gateway using a password or refresh token
	// host is the server address
	async connect(host: string, clientID: string) {
		// TODO: use discovery to determine port and path
		// since this runs with SSR on the server side, localhost should be able to connect
		if (host != '') {
			this.host = host;
		}

		let url = `wss://${this.host}:${this.port}/ws`;
		console.log('HubAPI.connect: url=', url);
		// TBD: should we use the server CA cert for MiM prevention?
		await global.hapiConnect(url, clientID, '', '', '', this.connectionStatusHandler.bind(this));
		this.conStat.clientID = clientID;
	}

	// disconnect if connected
	async disconnect() {
		if (this.conStat.status != 'disconnected') {
			await global.hapiDisconnect();
			this.conStat.status = 'disconnected';
			this.conStat.statusMessage = 'disconnected by user';
		}
	}

	// callback handler invoked when the connection status has changed
	connectionStatusHandler(isConnected: boolean) {
		console.info('onConnectHandler. Connected=', isConnected);
		if (isConnected) {
			console.log('HubAPI connected');
			this.conStat.status = 'connected';
			this.conStat.statusMessage = 'Connected to the Hub gateway. Login to authenticate.';
		} else {
			console.log('HubAPI disconnected');
			this.conStat.status = 'disconnected';
			this.conStat.statusMessage = 'Connection to Hub gateway is lost';
		}
	}

	// login to the Hub gateway with the client's loginID and password
	// If a refresh token already exists, use loginRefresh instead
	// This is the same login ID used in connect.
	// Returns the refresh token
	async login(clientID: string, password: string): Promise<string> {
		this.conStat.clientID = clientID;
		const refreshToken = await global.hapiLogin(clientID, password);
		if (refreshToken != '') {
			this.conStat.status = 'authenticated';
			this.conStat.statusMessage = 'Authenticated as ' + clientID;
		} else {
			this.conStat.status = 'connected';
			this.conStat.statusMessage = 'Authentication failed';
		}
		return refreshToken;
	}

	// loginRefresh refreshes a login using a valid refresh token
	// Returns a new refresh token
	async loginRefresh(clientID: string, refreshToken: string): Promise<string> {
		this.conStat.clientID = clientID;
		const newRefreshToken = await global.hapiLogin(clientID, refreshToken);
		if (newRefreshToken != '') {
			this.conStat.status = 'authenticated';
			this.conStat.statusMessage = 'Authenticated as ' + clientID;
		}
		return newRefreshToken;
	}

	// Publish a JSON encoded thing event
	async pubEvent(thingID: string, eventName: string, evJSON: string) {
		if (this.conStat.status == 'connected') {
			// defined in global.d.ts
			global.hapiPubEvent(thingID, eventName, evJSON);
		}
		return;
	}

	// Publish a Thing property map
	// Ignored if props map is empty
	async pubProperties(thingID: string, props: { [key: string]: any }) {
		// if (length(props.) > 0) {
		let propsJSON = JSON.stringify(props, null, ' ');
		if (propsJSON.length > 2) {
			this.pubEvent(thingID, EventTypes.Properties, propsJSON);
		}
	}

	// Publish a Thing TD document
	async pubTD(thingID: string, td: ThingTD) {
		let tdJSON = JSON.stringify(td, null, ' ');
		this.pubEvent(thingID, EventTypes.TD, tdJSON);
	}

	// Read Thing definitions from the directory
	// @param publisherID whose things to read or "" for all publishers
	// @param thingID whose to read or "" for all things of the publisher(s)
	async readDirectory(publisherID: string, thingID: string): Promise<string> {
		return global.hapiReadDirectory(publisherID, thingID);
	}

	// Subscribe to actions for things managed by this publisher.
	//
	// The 'actionID' is the key of the action in the TD action map,
	// or the hubapi.ActionNameConfiguration action which carries configuration key-value pairs.
	//
	// Authorization is handled by the message bus and not a concern of the service/device.
	//
	// Subscription requires a connection with the Hub. If the connection fails it must be
	// renewed.
	//
	// @param handler: handler of the action request, where:
	//  thingID: ID of the thing whose action is requested
	//  actionID: ID of the action requested as defined in the TD
	//  data: serialized event data
	async subActions(handler: (thingID: string, actionID: string, data: string) => void) {
		if (this.conStat.status == 'connected') {
			// defined in global.d.ts
			global.hapiSubActions(handler);
		}
		return;
	}

	// Subscribe to events from things
	//
	// @param publisherID: ID of the publisher whose things to subscribe to
	// @param thingID: ID of the thing to subscribe to or "" for any
	// @param eventID: ID of the event requested as defined in the TD or "" for any
	// @param handler: handler of the action request, where:
	//  publisherID: ID of the publisher sending the event
	//  thingID: ID of the thing sending the event
	//  eventID: ID of the event being sent
	//  data: serialized event data
	async subEvents(
		publisherID: string,
		thingID: string,
		eventID: string,
		handler: (publisherID: string, thingID: string, eventID: string, data: string) => void
	) {
		if (this.conStat.status == 'connected') {
			// defined in global.d.ts
			global.hapiSubEvents(publisherID, thingID, eventID, handler);
		}
		return;
	}
}

// 'hapi' is the singleton connecting to the Hub
export const hapi = new HubAPI('hiveoview');
