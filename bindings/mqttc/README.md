# MQTT client binding

This binding connects to an existing MQTT message bus to provide access to WoT Things available on the HiveOT hub. 

## Status

This binding is in the requirements phase.


## Feature Overview (under construction)

1. Follow the WoT MQTT specification
1. Act as a client for an existing MQTT broker
1. Specify the role of the MQTT client
1. Handle read/write property requests
1. Handle invoke-action requests received via the mqtt bus
2. Handle subscribe/unsubscribe event requests received via the mqtt bus
2. Handle observe/unobserve property requests received via the mqtt bus
1. Publish subscribed events onto the MQTT bus
1. Publish observed property changes onto the MQTT bus


