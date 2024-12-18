import {TD} from "@hivelib/things/TD";
import {ActionStatus} from "@hivelib/hubclient/ActionStatus";
import {IConsumerClient} from "@hivelib/hubclient/IConsumerClient";


// IAgentClient defines the interface of the hub agent transport.
export interface IAgentClient extends IConsumerClient {

    // PubEvent publishes an event style message without a response.
    // It returns as soon as delivery to the hub is confirmed.
    //
    // Events are published by agents using their native ID, not the digital twin ID.
    // The Hub outbox broadcasts this event using the digital twin ID.
    //
    //	@param thingID native thingID as provided by the agent
    //	@param key ID of the event
    //	@param payload to publish in native format as per TD
    //
    // This throws an error if the event cannot not be delivered to the hub
    pubEvent(thingID: string, key:string, payload: any): void

    // PubProgressUpdate [agent] sends a delivery progress update to the hub.
    // The hub will update the status of the action in the digital twin and
    // notify the original sender.
    //
    // Intended for agents that have processed an incoming action request asynchronously
    // and need to send an update on further progress.
    pubProgressUpdate(stat: ActionStatus):void

    // pubMultipleProperties agent updates multiple property values. (not for consumers)
    // It returns as soon as delivery to the hub is confirmed.
    //
    // @param thingID is the native thingID of the device (not including the digital twin ID)
    // @param propMap is the property key-value map to publish where value is their native format
    //
    // This throws an error if the message cannot not be delivered to the hub
    pubMultipleProperties(thingID: string, propMap: {[key:string]:any}): void

    // pubProperty agent updates a property value. (not for consumers)
    // It returns as soon as delivery to the hub is confirmed.
    //
    // @param thingID is the native thingID of the device (not including the digital twin ID)
    // @param name is the property name as published in the TD
    // @param value is the property value
    //
    // This throws an error if the message cannot not be delivered to the hub
    pubProperty(thingID: string, name: string, value: any): void

    // PubTD publishes an TD document event.
    // It returns as soon as delivery to the hub is confirmed.
    // This is intended for agents, not for consumers.
    //
    // @param td is the Thing Description document describing the Thing
    pubTD(td: TD): void

}
