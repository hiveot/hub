


// ThingValue contains a Thing event, action or property value
//
//	{
//	   "agentID": {string},
//	   "thingID": {string},
//	   "name": {string},
//	   "data": [{byte array}],
//	   "created": {int64},   // msec since epoc
//	}
export class ThingMessage extends Object {

    // ThingID or capabilityID of the thing itself
    public thingID: string = ""

    // Key of event, action or property as defined in the TD event/action map.
    public key: string = ""

    // Data in native format, as defined by the TD affordance DataSchema
    public data: any = ""

    // Timestamp the value was created in unix time, msec since Epoch Jan 1st,1970 00:00 utc
    public createdMSec: number = 0

    // Expiry time of the value in msec since epoc.
    // Events expire based on their update interval.
    // Actions expiry is used for queueing. 0 means the action expires immediately after receiving it and is not queued.
    //expiry: bigint

    // Sequence of the message from its creator. Intended to prevent replay attacks.
    //sequence: bigint

    // Type of message: MessageTypeAction, MessageTypeEvent;
    // This is required.
    public messageType: string = ""

    // The message ID set by the runtime
    public messageID:string = ""

    // senderID is the account ID of the agent, service or user sending the message
    // to the hub.
    // This is required and used in authorization of the sender and routing of messages.
    // The underlying protocol binding MUST set this to the authenticated client.
    public senderID: string = ""

}