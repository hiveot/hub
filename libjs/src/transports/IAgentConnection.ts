import {TD} from "@hivelib/wot/TD";
import {IConsumerConnection, RequestHandler} from "@hivelib/transports/IConsumerConnection";
import {ResponseMessage} from "@hivelib/transports/Messages";


// IAgentClient defines the interface for use by  hub agents.
// The hub agent is unique to hiveot and not supported in the wot specification.
export interface IAgentConnection extends IConsumerConnection {

    // PubEvent publishes an event style message without a response.
    // This is a convenience function that uses sendNotification.
    //
    // Events are published by agents using their native thingID, not the digital
    //  twin ID. The Hub outbox broadcasts this event using the digital twin ID.
    //
    //	@param thingID native thingID as provided by the agent
    //	@param key ID of the event
    //	@param payload to publish in native format as per TD
    //
    // This throws an error if the event cannot not be delivered to the hub
    pubEvent(thingID: string, key:string, payload: any): void

    // pubMultipleProperties agent updates multiple property values.
    // This is a convenience function that uses sendNotification.
    //
    // @param thingID is the native thingID of the device (not the digital twin ID)
    // @param propMap is the property key-value map to publish where value is their native format
    //
    // This throws an error if the message cannot not be delivered to the hub
    pubMultipleProperties(thingID: string, propMap: {[key:string]:any}): void

    // pubProperty agent updates a property value.
    // This is a convenience function that uses sendNotification.
    //
    // @param thingID is the native thingID of the device (not the digital twin ID)
    // @param name is the property name as published in the TD
    // @param value is the property value
    //
    // This throws an error if the message cannot not be delivered to the hub
    pubProperty(thingID: string, name: string, value: any): void

    // PubTD publishes an TD document event.
    // This is a convenience function that uses sendNotification.
    //
    // @param td is the Thing Description document describing the Thing
    pubTD(td: TD): void

    // sendResponse [agent] sends a response status message to the hub.
    // The hub will update the status of the action in the digital twin and
    // notify the original sender.
    //
    // Intended for agents that have processed an incoming requests asynchronously
    // and need to send an update on further progress.
    sendResponse(resp: ResponseMessage):void


    // Set the handler for incoming requests.
    // This replaces any previously set handler.
    setRequestHandler(handler: RequestHandler):void
}
