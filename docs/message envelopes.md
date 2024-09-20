# Message Envelopes

Considerations for supporting message envelopes in WoT interaction.

Problem: 
When sending request messages from consumer to things, there is currently no facility to asynchronously track the progress of delivery and the result of the request. Transport protocols such as MQTT-3, Websocket, SSE, HTTPS don't have a standard way of linking requests and responses.

Routing: There is no specification on how to address a Thing in a request or consumer in a response message. This is needed in asynchronous messaging via a Hub, Gateway or message bus.

Agents: When Things are reached through an intermediary 'agent' there is no specification on defining the Agent that acts on behalf of a Thing. When a Hub is used, sending an action to a Thing requires specifying the agent that represents the Thing. Thing IDs are only unique within the connection that publishes them.
 - expand thingID? 

Digital Twin: Digital twin Things are not the same thing as Things themselves. They can have a different IDs and support different protocols. In HiveOT a digital twin Thing ID has the agent prefix and messages are all between digital twin and the Thing/Consumer.


Proposed Solution:
Define a application level message envelope for the purpose of tracking delivery progress and linking async responses to their request. 

Add a reference in the protocol definition of the TD to the message envelope used.

Use a Thing dataschema to defined the message envelope fields or define this as part of the WoT standard.

Standardize these fields:
   * data: containing the message payload as per interaction affordance dataschema. [required]
   * name: name of the TD interaction affordance that describes the data format. [required]
     * How to differentiate between property, event, and action?
   * messageID: ID of the message for tracking progress. [optional] 
   * timestamp: time the message was created. Useful if the message delivery is delayed for devices that are asleep and handle message expiry. [optional]
   * expires: time the message expires. Useful for delayed delivery of short lived messages. [optional]
   * reply-to: Sender ID of the message sender. Intended for routing result messages directly to the sender instead of broadcasting. Similar to the inbox in MQTT-5. [optional]

Allow optional extra fields
   