# ISY99x Protocol Binding

ISY99 is a legacy gateway for the Insteon protocol. It is not longer produced and replaced by the ISY994. The ISY994 which also works with this publisher, albeit limited to what the ISY99 can do.

The intent of this binding implementation is to be able to use some basic devices that are commonly used, mostly insteon switches. As Insteon is a legacy protocol not much development effort is to be expected.

## Status

This binding is a bare bones functional POC that is able to identify the Insteon device and issue basic commands.

Limitations:
1. only basic configuration
2. only basic actions
3. only support for switches and dimmer nodes
4. uses polling instead of subscription to ISY events
5. no support for ISY scenes and programs
6. no support for ISY ZWave or UPB  
7. the vocabulary mapping of property/event/action names is inconsistent. This needs to be fixed in hiveot itself first. 

Above restrictions can be fixed with some development and a working ISY/PLM for testing. Unfortunately

## Dependencies

This binding requires the use of an ISY99x INSTEON gateway device and a PLM device modem for connecting to the power net.

## Configuration

Edit isy99.yaml with the ISY99 gateway address and login name/password. The gateway can also be configured through the gateway node 'gatewayAddress' configuration.

See config files in ./test as examples
