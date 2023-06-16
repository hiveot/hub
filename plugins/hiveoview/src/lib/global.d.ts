// TS definitions with Hub API global methods as provided by the wasm module

declare module globalThis {
  // wasm needs websocket access via globalThis
  // var WebSocket: any
}

declare global {
  // wasm needs websocket access via globalThis
  var WebSocket: any;

  function onGoStarted();

  // var Go: any;
  // Connect to the Hub
  function hapiConnect(
    url: string,
    clientID: string,
    certPem: string,
    keyPem: string,
    caCertPem: string,
    onConnectHandler: function(boolean)
  );

  // Disconnect released pubsub capabilities and disconnects from the Hub
  function hapiDisconnect();

  // Login to the hub with userID and password
  // This returns an updated refresh token for future logins/refresh
  function hapiLogin(userID: string, password: string): Promise<string>;

  // Login to the hub with userID and refresh token
  // This returns an updated refresh token for future logins/refresh
  function hapiLoginRefresh(userID: string, refreshToken: string): Promise<string>;

  // Publish an action
  function hapiPubAction(thingID: string, actionID: string, value: any);

  // Publish an event
  function hapiPubEvent(thingID: string, eventID: string, value: any);

  // Read directory
  function hapiReadDirectory(publisherID: string, thingID: string);

  // Subscribe to action requests
  function hapiSubActions(handler: (thingID: string, actionID: string, params: string) => void);

  // Subscribe to events
  function hapiSubEvents(
    pubID: string,
    thingID: string,
    eventID: string,
    handler: (pubID: string, thingID: string, eventID: string, params: string) => void
  );

  function gostop();
}

export {};

// declare module globalThis {

// }
