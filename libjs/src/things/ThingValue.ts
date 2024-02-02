


// ThingValue contains a Thing event, action or property value
//
//	{
//	   "agentID": {string},
//	   "thingID": {string},
//	   "name": {string},
//	   "data": [{byte array}],
//	   "created": {int64},   // msec since epoc
//	}
export class ThingValue extends Object {

    // AgentID is the ID of the device or service that owns the Thing
    public agentID: string = ""

    // ThingID or capabilityID of the thing itself
    public thingID: string = ""

    // Name of event, action or property as defined in the TD event/action map.
    public name: string = ""

    // Data with serialized value payload, as defined by the TD affordance DataSchema
    public data: string = ""

    // Timestamp the value was created in unix time, msec since Epoch Jan 1st,1970 00:00 utc
    public createdMSec: number = 0

    // Expiry time of the value in msec since epoc.
    // Events expire based on their update interval.
    // Actions expiry is used for queueing. 0 means the action expires immediately after receiving it and is not queued.
    //expiry: bigint

    // Sequence of the message from its creator. Intended to prevent replay attacks.
    //sequence: bigint

    // ID of the publisher of the value
    // For events this is the agentID
    // For actions,config and rpc this is the remote user sending the request
    public senderID: string = ""

    // Type of value, event, action, config, rpc
    public valueType: string = ""
}